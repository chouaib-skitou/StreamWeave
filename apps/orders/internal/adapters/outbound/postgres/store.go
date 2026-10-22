package postgres

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chouaib-skitou/streamweave/apps/orders/internal/adapters/outbound/postgres/generated"
	app "github.com/chouaib-skitou/streamweave/apps/orders/internal/application/orders"
	domain "github.com/chouaib-skitou/streamweave/apps/orders/internal/domain/orders"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sqlc-dev/pqtype"
)

type Store struct {
	db            *sql.DB
	q             *generated.Queries
	quoteVerifier *QuoteVerifier
	cursorSecret  []byte
}

type OutboxEvent struct {
	ID            uuid.UUID
	Topic         string
	PartitionKey  string
	MessageType   string
	AggregateType string
	AggregateID   string
	Payload       []byte
	Headers       []byte
	Attempts      int32
}

func NewStore(connection *Connection) *Store {
	return NewStoreWithQuoteVerifierAndCursor(connection, nil, "")
}

func NewStoreWithQuoteVerifier(connection *Connection, verifier *QuoteVerifier) *Store {
	return NewStoreWithQuoteVerifierAndCursor(connection, verifier, "")
}

func NewStoreWithQuoteVerifierAndCursor(connection *Connection, verifier *QuoteVerifier, cursorSecret string) *Store {
	return &Store{db: connection.DB(), q: generated.New(connection.DB()), quoteVerifier: verifier, cursorSecret: []byte(cursorSecret)}
}

func (s *Store) Check(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *Store) FindByID(ctx context.Context, orderID string) (domain.Order, error) {
	return loadOrder(ctx, s.q, orderID)
}

func (s *Store) Create(ctx context.Context, actorID string, order domain.Order, operation, key, requestHash string) (app.CreateResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return app.CreateResult{}, mapError(err)
	}
	q := generated.New(tx)
	defer func() { _ = tx.Rollback() }()

	accepted := acceptedResponse{OrderID: order.ID, Status: string(order.Status), CreatedAt: order.CreatedAt}
	body, err := json.Marshal(accepted)
	if err != nil {
		return app.CreateResult{}, err
	}
	inserted, err := q.InsertIdempotency(ctx, generated.InsertIdempotencyParams{
		ActorID: actorID, Operation: operation, Key: key, RequestHash: requestHash,
		ResourceID: sql.NullString{String: order.ID, Valid: true}, ResponseStatus: sql.NullInt32{Int32: 202, Valid: true},
		ResponseBody: pqtype.NullRawMessage{RawMessage: body, Valid: true}, CreatedAt: order.CreatedAt, ExpiresAt: order.CreatedAt.Add(48 * time.Hour),
	})
	if err != nil {
		return app.CreateResult{}, mapError(err)
	}
	if inserted == 0 {
		record, getErr := q.GetIdempotency(ctx, generated.GetIdempotencyParams{ActorID: actorID, Operation: operation, Key: key})
		if getErr != nil {
			return app.CreateResult{}, mapError(getErr)
		}
		if record.RequestHash != requestHash {
			return app.CreateResult{}, app.ErrIdempotencyConflict
		}
		replayed, getErr := loadOrder(ctx, q, record.ResourceID.String)
		if getErr != nil {
			return app.CreateResult{}, mapError(getErr)
		}
		if err := tx.Commit(); err != nil {
			return app.CreateResult{}, mapError(err)
		}
		accepted := acceptedResponse{OrderID: replayed.ID, Status: string(replayed.Status), CreatedAt: replayed.CreatedAt}
		if record.ResponseBody.Valid {
			_ = json.Unmarshal(record.ResponseBody.RawMessage, &accepted)
		}
		return app.CreateResult{Order: replayed, Replayed: true, ResponseStatus: accepted.Status, ResponseAt: accepted.CreatedAt}, nil
	}

	if err := q.InsertOrder(ctx, generated.InsertOrderParams{ID: order.ID, CustomerID: order.CustomerID, Currency: order.Currency, Status: string(order.Status), AggregateVersion: int64(order.Version), CreatedAt: order.CreatedAt}); err != nil {
		return app.CreateResult{}, mapError(err)
	}
	for index, item := range order.Items {
		if err := q.InsertOrderItem(ctx, generated.InsertOrderItemParams{OrderID: order.ID, LineNumber: int32(index + 1), ProductID: item.ProductID, Quantity: item.Quantity}); err != nil {
			return app.CreateResult{}, mapError(err)
		}
	}
	sagaID := "sag_" + uuid.NewString()
	if err := q.InsertSaga(ctx, generated.InsertSagaParams{ID: sagaID, OrderID: order.ID, Phase: "INVENTORY_RESERVATION_PENDING", CreatedAt: order.CreatedAt}); err != nil {
		return app.CreateResult{}, mapError(err)
	}
	createdEventID := "evt_" + uuid.NewString()
	payload, err := eventPayloadWithID(createdEventID, "commerce.order.created.v1", order, sagaID, "cmd_"+uuid.NewString(), "ord_"+strings.TrimPrefix(order.ID, "ord_")+":create", "")
	if err != nil {
		return app.CreateResult{}, err
	}
	if err := q.InsertOutbox(ctx, generated.InsertOutboxParams{ID: uuid.New(), Topic: "commerce.order.events.v1", PartitionKey: order.ID, MessageType: "commerce.order.created.v1", AggregateType: "order", AggregateID: order.ID, Payload: payload, Headers: []byte(`{"content-type":"application/json"}`), CreatedAt: order.CreatedAt}); err != nil {
		return app.CreateResult{}, mapError(err)
	}
	reserve, err := commandPayload("commerce.inventory.reserve.v1", order, sagaID, createdEventID, order.ID+":reserve", map[string]any{
		"order_id": order.ID, "customer_id": order.CustomerID, "currency": order.Currency, "items": orderItemsPayload(order),
	})
	if err != nil {
		return app.CreateResult{}, err
	}
	if err := insertOutbox(ctx, q, "commerce.inventory.reserve.v1", order.ID, reserve, order.CreatedAt); err != nil {
		return app.CreateResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return app.CreateResult{}, mapError(err)
	}
	return app.CreateResult{Order: order, ResponseStatus: string(order.Status), ResponseAt: order.CreatedAt}, nil
}

func (s *Store) List(ctx context.Context, customerID, status, cursor string, limit int) ([]domain.Order, string, error) {
	var c cursorValue
	if cursor != "" {
		decoded, err := decodeSignedCursor(cursor, s.cursorSecret)
		if err != nil || decoded.CustomerID != customerID || decoded.Status != status {
			return nil, "", app.ErrInvalidCursor
		}
		c = decoded
	}
	rows, err := s.q.ListOrders(ctx, generated.ListOrdersParams{CustomerID: customerID, Column2: status, CursorCreatedAt: nullTime(c.CreatedAt), CursorID: nullString(c.ID), PageSize: int32(limit + 1)})
	if err != nil {
		return nil, "", mapError(err)
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	orders := make([]domain.Order, 0, len(rows))
	for _, row := range rows {
		order, loadErr := loadOrder(ctx, s.q, row.ID)
		if loadErr != nil {
			return nil, "", loadErr
		}
		orders = append(orders, order)
	}
	if !hasMore || len(orders) == 0 {
		return orders, "", nil
	}
	last := orders[len(orders)-1]
	next, err := encodeSignedCursor(cursorValue{CustomerID: customerID, Status: status, CreatedAt: last.CreatedAt, ID: last.ID}, s.cursorSecret)
	return orders, next, err
}

func (s *Store) RequestCancellation(ctx context.Context, actorID, orderID, operation, key, requestHash, reason string, expectedVersion uint64, at time.Time) (domain.Order, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Order{}, mapError(err)
	}
	q := generated.New(tx)
	defer func() { _ = tx.Rollback() }()
	inserted, err := q.InsertIdempotency(ctx, generated.InsertIdempotencyParams{ActorID: actorID, Operation: operation, Key: key, RequestHash: requestHash, ResponseStatus: sql.NullInt32{Int32: 202, Valid: true}, CreatedAt: at, ExpiresAt: at.Add(48 * time.Hour)})
	if err != nil {
		return domain.Order{}, mapError(err)
	}
	if inserted == 0 {
		record, getErr := q.GetIdempotency(ctx, generated.GetIdempotencyParams{ActorID: actorID, Operation: operation, Key: key})
		if getErr != nil {
			return domain.Order{}, mapError(getErr)
		}
		if record.RequestHash != requestHash {
			return domain.Order{}, app.ErrIdempotencyConflict
		}
		order, getErr := loadOrder(ctx, q, record.ResourceID.String)
		if getErr != nil {
			return domain.Order{}, getErr
		}
		if err := tx.Commit(); err != nil {
			return domain.Order{}, mapError(err)
		}
		return order, nil
	}
	row, err := q.UpdateOrderCancellation(ctx, generated.UpdateOrderCancellationParams{ID: orderID, AggregateVersion: int64(expectedVersion), UpdatedAt: at})
	if err != nil {
		return domain.Order{}, mapError(err)
	}
	saga, err := q.GetSagaByOrderID(ctx, orderID)
	if err != nil {
		return domain.Order{}, mapError(err)
	}
	sagaID := saga.ID
	eventID := "evt_" + uuid.NewString()
	accepted := acceptedResponse{OrderID: row.ID, Status: row.Status, CreatedAt: row.UpdatedAt}
	body, _ := json.Marshal(accepted)
	if err = q.UpdateIdempotencyResponse(ctx, generated.UpdateIdempotencyResponseParams{ActorID: actorID, Operation: operation, Key: key, ResourceID: sql.NullString{String: row.ID, Valid: true}, ResponseBody: pqtype.NullRawMessage{RawMessage: body, Valid: true}}); err != nil {
		return domain.Order{}, mapError(err)
	}
	payload, err := eventPayloadWithID(eventID, "commerce.order.cancellation_requested.v1", cancellationRowToOrder(row), sagaID, "cmd_"+uuid.NewString(), "ord_"+strings.TrimPrefix(orderID, "ord_")+":cancel", reason)
	if err != nil {
		return domain.Order{}, err
	}
	if err := q.InsertOutbox(ctx, generated.InsertOutboxParams{ID: uuid.New(), Topic: "commerce.order.events.v1", PartitionKey: orderID, MessageType: "commerce.order.cancellation_requested.v1", AggregateType: "order", AggregateID: orderID, Payload: payload, Headers: []byte(`{"reason_code":"` + reason + `"}`), CreatedAt: at}); err != nil {
		return domain.Order{}, mapError(err)
	}
	if err := q.UpdateSagaPhase(ctx, generated.UpdateSagaPhaseParams{ID: sagaID, Phase: "CANCELLATION_REQUESTED", UpdatedAt: at}); err != nil {
		return domain.Order{}, mapError(err)
	}
	if err := s.insertCompensationCommands(ctx, q, cancellationRowToOrder(row), sagaID, eventID, "commerce.order.cancellation_requested.v1", at); err != nil {
		return domain.Order{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Order{}, mapError(err)
	}
	return cancellationRowToOrder(row), nil
}

func (s *Store) PendingOutbox(ctx context.Context, limit int) ([]OutboxEvent, error) {
	rows, err := s.q.ListPendingOutbox(ctx, int32(limit))
	if err != nil {
		return nil, mapError(err)
	}
	result := make([]OutboxEvent, 0, len(rows))
	for _, row := range rows {
		result = append(result, OutboxEvent{ID: row.ID, Topic: row.Topic, PartitionKey: row.PartitionKey, MessageType: row.MessageType, AggregateType: row.AggregateType, AggregateID: row.AggregateID, Payload: row.Payload, Headers: row.Headers, Attempts: row.Attempts})
	}
	return result, nil
}

// ClaimPendingOutbox leases a bounded batch so multiple relay replicas cannot publish the same row concurrently.
func (s *Store) ClaimPendingOutbox(ctx context.Context, owner string, limit int, until time.Time) ([]OutboxEvent, error) {
	rows, err := s.q.ClaimPendingOutbox(ctx, generated.ClaimPendingOutboxParams{LeaseOwner: sql.NullString{String: owner, Valid: true}, LeaseUntil: sql.NullTime{Time: until, Valid: true}, Limit: int32(limit)})
	if err != nil {
		return nil, mapError(err)
	}
	result := make([]OutboxEvent, 0, len(rows))
	for _, row := range rows {
		result = append(result, OutboxEvent{ID: row.ID, Topic: row.Topic, PartitionKey: row.PartitionKey, MessageType: row.MessageType, AggregateType: row.AggregateType, AggregateID: row.AggregateID, Payload: row.Payload, Headers: row.Headers, Attempts: row.Attempts})
	}
	return result, nil
}

func (s *Store) MarkOutboxPublished(ctx context.Context, id uuid.UUID, at time.Time) error {
	return mapError(s.q.MarkOutboxPublished(ctx, generated.MarkOutboxPublishedParams{ID: id, PublishedAt: sql.NullTime{Time: at, Valid: true}}))
}
func (s *Store) MarkOutboxFailed(ctx context.Context, id uuid.UUID, reason string, retryAt time.Time) error {
	return mapError(s.q.MarkOutboxFailed(ctx, generated.MarkOutboxFailedParams{ID: id, LastError: sql.NullString{String: reason, Valid: true}, NextAttemptAt: retryAt}))
}
func (s *Store) PendingOutboxCount(ctx context.Context) (int64, error) {
	count, err := s.q.CountPendingOutbox(ctx)
	return count, mapError(err)
}

// RecordFactFailure advances durable Inbox retry state and reports when the message is quarantined.
func (s *Store) RecordFactFailure(ctx context.Context, consumerGroup string, payload []byte, reason string, retryAt time.Time) (bool, int32, error) {
	var envelope factEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil || envelope.EventID == "" || envelope.EventType == "" || envelope.CorrelationID == "" {
		// The payload has no durable Inbox identity; the relay must quarantine it immediately.
		return true, 0, nil
	}
	hash := sha256.Sum256(payload)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, 0, mapError(err)
	}
	q := generated.New(tx)
	defer func() { _ = tx.Rollback() }()
	if len(reason) > 512 {
		reason = reason[:512]
	}
	row, err := q.RecordInboxFailure(ctx, generated.RecordInboxFailureParams{
		ConsumerGroup: consumerGroup,
		MessageID:     envelope.EventID,
		MessageType:   envelope.EventType,
		PayloadHash:   fmt.Sprintf("%x", hash[:]),
		ReceivedAt:    time.Now().UTC(),
		CorrelationID: envelope.CorrelationID,
		CausationID:   nullString(envelope.CausationID),
		LastError:     sql.NullString{String: reason, Valid: true},
		NextAttemptAt: sql.NullTime{Time: retryAt, Valid: true},
	})
	if err != nil {
		return false, 0, mapError(err)
	}
	if err := tx.Commit(); err != nil {
		return false, 0, mapError(err)
	}
	return row.Attempts >= 8, row.Attempts, nil
}

func (s *Store) MarkFactDeadLetter(ctx context.Context, consumerGroup, messageID, reason string) error {
	return mapError(s.q.MarkInboxDeadLetter(ctx, generated.MarkInboxDeadLetterParams{ConsumerGroup: consumerGroup, MessageID: messageID, LastError: sql.NullString{String: reason, Valid: true}}))
}

type factEnvelope struct {
	EventID          string                     `json:"event_id"`
	EventType        string                     `json:"event_type"`
	AggregateID      string                     `json:"aggregate_id"`
	AggregateType    string                     `json:"aggregate_type"`
	AggregateVersion uint64                     `json:"aggregate_version"`
	SagaID           string                     `json:"saga_id"`
	CorrelationID    string                     `json:"correlation_id"`
	CausationID      string                     `json:"causation_id"`
	OperationID      string                     `json:"operation_id"`
	Producer         string                     `json:"producer"`
	SchemaVersion    int                        `json:"schema_version"`
	OccurredAt       time.Time                  `json:"occurred_at"`
	Data             map[string]json.RawMessage `json:"data"`
}

// ProcessFact applies one dependency fact and records its Inbox identity in the same transaction.
func (s *Store) ProcessFact(ctx context.Context, consumerGroup string, payload []byte) error {
	var envelope factEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil || envelope.EventID == "" || envelope.AggregateID == "" || envelope.EventType == "" || envelope.AggregateType == "" || envelope.Producer == "" || envelope.SchemaVersion != 1 {
		return fmt.Errorf("%w: invalid event envelope", app.ErrConflict)
	}
	orderID := rawString(envelope.Data["order_id"])
	if !domain.ValidOrderID(orderID) || envelope.CorrelationID != orderID || !validFactIdentity(envelope) {
		return fmt.Errorf("%w: invalid fact identity", app.ErrConflict)
	}
	hash := sha256.Sum256(payload)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return mapError(err)
	}
	q := generated.New(tx)
	defer func() { _ = tx.Rollback() }()
	inserted, err := q.InsertInbox(ctx, generated.InsertInboxParams{ConsumerGroup: consumerGroup, MessageID: envelope.EventID, MessageType: envelope.EventType, PayloadHash: fmt.Sprintf("%x", hash[:]), ReceivedAt: time.Now().UTC(), CorrelationID: envelope.CorrelationID, CausationID: nullString(envelope.CausationID)})
	if err != nil {
		return mapError(err)
	}
	if inserted == 0 {
		record, getErr := q.GetInbox(ctx, generated.GetInboxParams{ConsumerGroup: consumerGroup, MessageID: envelope.EventID})
		if getErr != nil {
			return mapError(getErr)
		}
		if record.Status == "PROCESSED" || record.Status == "DLQ" {
			return nil
		}
	}
	order, err := loadOrder(ctx, q, orderID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	switch envelope.EventType {
	case "commerce.inventory.reserved.v1":
		if err := s.applyInventoryReserved(ctx, q, order, envelope, now); err != nil {
			return err
		}
	case "commerce.inventory.released.v1":
		if err := s.applyInventoryReleased(ctx, q, order, envelope, now); err != nil {
			return err
		}
	case "commerce.payment.refunded.v1":
		if err := s.applyPaymentRefunded(ctx, q, order, envelope, now); err != nil {
			return err
		}
	case "commerce.inventory.rejected.v1", "commerce.payment.failed.v1", "commerce.fulfillment.failed.v1":
		if order.Status == domain.StatusCancelReq {
			if envelope.EventType == "commerce.inventory.rejected.v1" {
				if err := s.markInventoryReleased(ctx, q, &order, now); err != nil {
					return err
				}
			} else if envelope.EventType == "commerce.payment.failed.v1" {
				if err := s.markPaymentRefunded(ctx, q, &order, now); err != nil {
					return err
				}
			}
			if err := s.insertCompensationCommands(ctx, q, order, envelope.SagaID, envelope.EventID, envelope.EventType, now); err != nil {
				return err
			}
			if err := s.maybeCompleteCancellation(ctx, q, order, envelope.SagaID, envelope.EventID, now); err != nil {
				return err
			}
			break
		}
		if order.Status.IsTerminal() {
			break
		}
		reason := rawString(envelope.Data["reason_code"])
		if reason == "" {
			reason = "DOWNSTREAM_REJECTED"
		}
		if err := s.updateWorkflow(ctx, q, order, domain.StatusFailed, reason, "", "", "", order.TotalMinor, now); err != nil {
			return err
		}
		order.Version++
		failed, buildErr := eventPayloadFor("commerce.order.failed.v1", order, envelope.SagaID, envelope.EventID, order.ID+":failed", map[string]any{"order_id": order.ID, "reason_code": reason, "failure_class": "dependency"})
		if buildErr != nil {
			return buildErr
		}
		if err := insertOutbox(ctx, q, "commerce.order.failed.v1", order.ID, failed, now); err != nil {
			return err
		}
		if err := s.insertCompensationCommands(ctx, q, order, envelope.SagaID, envelope.EventID, envelope.EventType, now); err != nil {
			return err
		}
	case "commerce.payment.authorized.v1":
		if err := s.applyPaymentAuthorized(ctx, q, order, envelope, now); err != nil {
			return err
		}
	case "commerce.fulfillment.completed.v1":
		if order.Status.IsTerminal() {
			break
		}
		shipmentID := rawString(envelope.Data["shipment_id"])
		if !domain.ValidShipmentID(shipmentID) {
			return fmt.Errorf("%w: shipment reference missing", app.ErrConflict)
		}
		if err := s.updateWorkflow(ctx, q, order, domain.StatusCompleted, "", "", "", shipmentID, order.TotalMinor, now); err != nil {
			return err
		}
		order.Version++
		payload, buildErr := eventPayloadFor("commerce.order.completed.v1", order, envelope.SagaID, envelope.EventID, order.ID+":complete", map[string]any{"order_id": order.ID, "shipment_id": shipmentID})
		if buildErr != nil {
			return buildErr
		}
		if err := insertOutbox(ctx, q, "commerce.order.completed.v1", order.ID, payload, now); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: unsupported event %s", app.ErrConflict, envelope.EventType)
	}
	if err := q.MarkInboxProcessed(ctx, generated.MarkInboxProcessedParams{ConsumerGroup: consumerGroup, MessageID: envelope.EventID, ProcessedAt: sql.NullTime{Time: now, Valid: true}}); err != nil {
		return mapError(err)
	}
	if err := tx.Commit(); err != nil {
		return mapError(err)
	}
	return nil
}

func (s *Store) insertCompensationCommands(ctx context.Context, q *generated.Queries, order domain.Order, sagaID, causation, failedEvent string, at time.Time) error {
	if (failedEvent == "commerce.payment.failed.v1" || failedEvent == "commerce.fulfillment.failed.v1" || failedEvent == "commerce.order.cancellation_requested.v1") && order.ReservationID != nil && domain.ValidReservationID(*order.ReservationID) {
		release, err := commandPayload("commerce.inventory.release.v1", order, sagaID, causation, order.ID+":release-inventory", map[string]any{"order_id": order.ID, "reservation_id": *order.ReservationID})
		if err != nil {
			return err
		}
		if err := insertOutbox(ctx, q, "commerce.inventory.release.v1", order.ID, release, at); err != nil {
			return err
		}
	}
	if (failedEvent == "commerce.fulfillment.failed.v1" || failedEvent == "commerce.order.cancellation_requested.v1") && order.PaymentID != nil && domain.ValidPaymentID(*order.PaymentID) && order.TotalMinor != nil {
		refund, err := commandPayload("commerce.payment.refund.v1", order, sagaID, causation, order.ID+":refund-payment", map[string]any{"order_id": order.ID, "payment_id": *order.PaymentID, "amount_minor": *order.TotalMinor, "currency": order.Currency})
		if err != nil {
			return err
		}
		return insertOutbox(ctx, q, "commerce.payment.refund.v1", order.ID, refund, at)
	}
	return nil
}

func (s *Store) applyInventoryReserved(ctx context.Context, q *generated.Queries, order domain.Order, envelope factEnvelope, now time.Time) error {
	if order.Status.IsTerminal() {
		return nil
	}
	reservationID := rawString(envelope.Data["reservation_id"])
	if !domain.ValidReservationID(reservationID) {
		return fmt.Errorf("%w: reservation reference is invalid", app.ErrConflict)
	}
	if envelope.OperationID != order.ID+":reserve" {
		return fmt.Errorf("%w: reservation operation mismatch", app.ErrConflict)
	}
	if order.Status == domain.StatusCancelReq {
		order.ReservationID = &reservationID
		if err := s.updateWorkflow(ctx, q, order, domain.StatusCancelReq, "", reservationID, "", "", order.TotalMinor, now); err != nil {
			return err
		}
		order.Version++
		return s.insertCompensationCommands(ctx, q, order, envelope.SagaID, envelope.EventID, "commerce.order.cancellation_requested.v1", now)
	}
	if order.Status != domain.StatusPending {
		return nil
	}
	expectedVersion := order.Version
	currency := rawString(envelope.Data["currency"])
	if currency != order.Currency {
		return fmt.Errorf("%w: quote currency mismatch", app.ErrConflict)
	}
	quoteID := rawString(envelope.Data["quote_id"])
	quoteVersion, err := rawInt64(envelope.Data["quote_version"])
	if err != nil {
		return err
	}
	if envelope.OperationID != order.ID+":reserve" {
		return fmt.Errorf("%w: reservation operation mismatch", app.ErrConflict)
	}
	quoteExpiresAt := rawString(envelope.Data["quote_expires_at"])
	quote := signedQuote{OrderID: order.ID, ReservationID: reservationID, QuoteID: quoteID, QuoteVersion: quoteVersion, Currency: currency, QuoteExpiresAt: quoteExpiresAt}
	var lines []quoteItem
	if err := json.Unmarshal(envelope.Data["items"], &lines); err != nil || len(lines) != len(order.Items) {
		return fmt.Errorf("%w: quote items mismatch", app.ErrConflict)
	}
	quote.Items = lines
	if err := s.quoteVerifier.Verify(ctx, quote, rawString(envelope.Data["quote_key_id"]), rawString(envelope.Data["quote_signature"]), now); err != nil {
		return err
	}
	prices := make(map[string]int64, len(lines))
	quantities := make(map[string]int32, len(lines))
	for _, line := range lines {
		if !domain.ValidProductID(line.ProductID) || line.Quantity < 1 || line.UnitPriceMinor < 0 || line.LineTotalMinor < 0 {
			return fmt.Errorf("%w: invalid quote line", app.ErrConflict)
		}
		if _, exists := prices[line.ProductID]; exists {
			return fmt.Errorf("%w: duplicate quote product", app.ErrConflict)
		}
		if line.UnitPriceMinor != 0 && int64(line.Quantity) > int64(1<<63-1)/line.UnitPriceMinor {
			return fmt.Errorf("%w: quote line overflow", app.ErrConflict)
		}
		if line.LineTotalMinor != line.UnitPriceMinor*int64(line.Quantity) {
			return fmt.Errorf("%w: quote line total mismatch", app.ErrConflict)
		}
		prices[line.ProductID] = line.UnitPriceMinor
		quantities[line.ProductID] = line.Quantity
	}
	for _, item := range order.Items {
		if quantities[item.ProductID] != item.Quantity {
			return fmt.Errorf("%w: quote quantity mismatch", app.ErrConflict)
		}
	}
	if err := order.ApplyQuote(quoteID, quoteVersion, prices, now); err != nil {
		return err
	}
	for _, item := range order.Items {
		if err := q.UpdateOrderItemQuote(ctx, generated.UpdateOrderItemQuoteParams{OrderID: order.ID, ProductID: item.ProductID, UnitPriceMinor: sql.NullInt64{Int64: *item.UnitPriceMinor, Valid: true}, LineTotalMinor: sql.NullInt64{Int64: *item.LineTotalMinor, Valid: true}, QuoteID: sql.NullString{String: *item.QuoteID, Valid: true}, QuoteVersion: sql.NullInt64{Int64: *item.QuoteVersion, Valid: true}}); err != nil {
			return mapError(err)
		}
	}
	order.Version = expectedVersion
	if err := s.updateWorkflow(ctx, q, order, domain.StatusPending, "", reservationID, "", "", order.TotalMinor, now); err != nil {
		return err
	}
	order.Version++
	order.ReservationID = &reservationID
	command, err := commandPayload("commerce.payment.authorize.v1", order, envelope.SagaID, envelope.EventID, order.ID+":authorize-payment", map[string]any{"order_id": order.ID, "amount_minor": *order.TotalMinor, "currency": order.Currency})
	if err != nil {
		return err
	}
	return insertOutbox(ctx, q, "commerce.payment.authorize.v1", order.ID, command, now)
}

func (s *Store) applyPaymentAuthorized(ctx context.Context, q *generated.Queries, order domain.Order, envelope factEnvelope, now time.Time) error {
	if order.Status.IsTerminal() || order.Status == domain.StatusConfirmed {
		return nil
	}
	amount, err := rawInt64(envelope.Data["amount_minor"])
	if err != nil || order.TotalMinor == nil || amount != *order.TotalMinor || rawString(envelope.Data["currency"]) != order.Currency {
		return fmt.Errorf("%w: payment amount mismatch", app.ErrConflict)
	}
	paymentID := rawString(envelope.Data["payment_id"])
	if !domain.ValidPaymentID(paymentID) {
		return fmt.Errorf("%w: payment reference missing", app.ErrConflict)
	}
	if order.Status == domain.StatusCancelReq {
		order.PaymentID = &paymentID
		if err := s.updateWorkflow(ctx, q, order, domain.StatusCancelReq, "", "", paymentID, "", order.TotalMinor, now); err != nil {
			return err
		}
		order.Version++
		return s.insertCompensationCommands(ctx, q, order, envelope.SagaID, envelope.EventID, "commerce.order.cancellation_requested.v1", now)
	}
	if err := s.updateWorkflow(ctx, q, order, domain.StatusConfirmed, "", "", paymentID, "", order.TotalMinor, now); err != nil {
		return err
	}
	order.Version++
	order.PaymentID = &paymentID
	if order.ReservationID == nil || *order.ReservationID == "" {
		return fmt.Errorf("%w: reservation reference missing", app.ErrConflict)
	}
	confirmed, err := eventPayloadFor("commerce.order.confirmed.v1", order, envelope.SagaID, envelope.EventID, order.ID+":confirm", map[string]any{"order_id": order.ID, "payment_id": paymentID, "reservation_id": *order.ReservationID})
	if err != nil {
		return err
	}
	if err := insertOutbox(ctx, q, "commerce.order.confirmed.v1", order.ID, confirmed, now); err != nil {
		return err
	}
	command, err := commandPayload("commerce.fulfillment.create.v1", order, envelope.SagaID, envelope.EventID, order.ID+":create-fulfillment", map[string]any{
		"order_id": order.ID, "customer_id": order.CustomerID, "reservation_id": *order.ReservationID, "currency": order.Currency, "items": orderItemsPayload(order),
	})
	if err != nil {
		return err
	}
	return insertOutbox(ctx, q, "commerce.fulfillment.create.v1", order.ID, command, now)
}

func (s *Store) applyInventoryReleased(ctx context.Context, q *generated.Queries, order domain.Order, envelope factEnvelope, now time.Time) error {
	if order.ReservationID == nil || rawString(envelope.Data["reservation_id"]) != *order.ReservationID {
		return fmt.Errorf("%w: reservation reference mismatch", app.ErrConflict)
	}
	if order.Status.IsTerminal() && order.InventoryReleased {
		return nil
	}
	if err := s.markInventoryReleased(ctx, q, &order, now); err != nil {
		return err
	}
	return s.maybeCompleteCancellation(ctx, q, order, envelope.SagaID, envelope.EventID, now)
}

func (s *Store) applyPaymentRefunded(ctx context.Context, q *generated.Queries, order domain.Order, envelope factEnvelope, now time.Time) error {
	if order.PaymentID == nil || rawString(envelope.Data["payment_id"]) != *order.PaymentID {
		return fmt.Errorf("%w: payment reference mismatch", app.ErrConflict)
	}
	if order.Status.IsTerminal() && order.PaymentRefunded {
		return nil
	}
	if err := s.markPaymentRefunded(ctx, q, &order, now); err != nil {
		return err
	}
	return s.maybeCompleteCancellation(ctx, q, order, envelope.SagaID, envelope.EventID, now)
}

func (s *Store) markInventoryReleased(ctx context.Context, q *generated.Queries, order *domain.Order, now time.Time) error {
	if order.InventoryReleased {
		return nil
	}
	rows, err := q.MarkInventoryReleased(ctx, generated.MarkInventoryReleasedParams{ID: order.ID, UpdatedAt: now, AggregateVersion: int64(order.Version)})
	if err != nil {
		return mapError(err)
	}
	if rows != 1 {
		return app.ErrConflict
	}
	order.InventoryReleased = true
	order.Version++
	order.UpdatedAt = now
	return nil
}

func (s *Store) markPaymentRefunded(ctx context.Context, q *generated.Queries, order *domain.Order, now time.Time) error {
	if order.PaymentRefunded {
		return nil
	}
	rows, err := q.MarkPaymentRefunded(ctx, generated.MarkPaymentRefundedParams{ID: order.ID, UpdatedAt: now, AggregateVersion: int64(order.Version)})
	if err != nil {
		return mapError(err)
	}
	if rows != 1 {
		return app.ErrConflict
	}
	order.PaymentRefunded = true
	order.Version++
	order.UpdatedAt = now
	return nil
}

func (s *Store) maybeCompleteCancellation(ctx context.Context, q *generated.Queries, order domain.Order, sagaID, causation string, now time.Time) error {
	if order.Status != domain.StatusCancelReq {
		return nil
	}
	if (order.ReservationID != nil && !order.InventoryReleased) || (order.PaymentID != nil && !order.PaymentRefunded) {
		return nil
	}
	if err := s.updateWorkflow(ctx, q, order, domain.StatusCancelled, "", "", "", "", order.TotalMinor, now); err != nil {
		return err
	}
	order.Version++
	payload, err := eventPayloadFor("commerce.order.cancelled.v1", order, sagaID, causation, order.ID+":cancelled", map[string]any{"order_id": order.ID, "reason_code": "CUSTOMER_REQUESTED"})
	if err != nil {
		return err
	}
	if err := insertOutbox(ctx, q, "commerce.order.cancelled.v1", order.ID, payload, now); err != nil {
		return err
	}
	return mapError(q.UpdateSagaPhase(ctx, generated.UpdateSagaPhaseParams{ID: sagaID, Phase: "CANCELLED", UpdatedAt: now}))
}

func (s *Store) updateWorkflow(ctx context.Context, q *generated.Queries, order domain.Order, status domain.Status, failure, reservation, payment, shipment string, total *int64, now time.Time) error {
	return s.updateWorkflowAtVersion(ctx, q, order, status, failure, reservation, payment, shipment, total, order.Version, now)
}

func (s *Store) updateWorkflowAtVersion(ctx context.Context, q *generated.Queries, order domain.Order, status domain.Status, failure, reservation, payment, shipment string, total *int64, expectedVersion uint64, now time.Time) error {
	if status != order.Status && order.Status.IsTerminal() {
		return nil
	}
	rows, err := q.UpdateOrderWorkflowState(ctx, generated.UpdateOrderWorkflowStateParams{ID: order.ID, Status: string(status), FailureCode: nullString(failure), ReservationID: nullString(reservation), PaymentID: nullString(payment), ShipmentID: nullString(shipment), TotalMinor: int64Null(total), UpdatedAt: now, AggregateVersion: int64(expectedVersion)})
	if err != nil {
		return mapError(err)
	}
	if rows != 1 {
		return app.ErrConflict
	}
	return nil
}

func validFactIdentity(envelope factEnvelope) bool {
	switch envelope.EventType {
	case "commerce.inventory.reserved.v1", "commerce.inventory.rejected.v1", "commerce.inventory.released.v1":
		return envelope.AggregateType == "inventory" && envelope.Producer == "inventory"
	case "commerce.payment.authorized.v1", "commerce.payment.failed.v1", "commerce.payment.refunded.v1":
		return envelope.AggregateType == "payment" && envelope.Producer == "payments"
	case "commerce.fulfillment.completed.v1", "commerce.fulfillment.failed.v1":
		return envelope.AggregateType == "fulfillment" && envelope.Producer == "fulfillment"
	default:
		return false
	}
}

func insertOutbox(ctx context.Context, q *generated.Queries, messageType, orderID string, payload []byte, at time.Time) error {
	return mapError(q.InsertOutbox(ctx, generated.InsertOutboxParams{ID: uuid.New(), Topic: topicFor(messageType), PartitionKey: orderID, MessageType: messageType, AggregateType: "order", AggregateID: orderID, Payload: payload, Headers: []byte(`{"content-type":"application/json"}`), CreatedAt: at}))
}
func topicFor(messageType string) string {
	if strings.HasPrefix(messageType, "commerce.order.") {
		return "commerce.order.events.v1"
	}
	if strings.HasPrefix(messageType, "commerce.inventory.") {
		return "commerce.inventory.commands.v1"
	}
	if strings.HasPrefix(messageType, "commerce.payment.") {
		return "commerce.payment.commands.v1"
	}
	return "commerce.fulfillment.commands.v1"
}
func commandPayload(commandType string, order domain.Order, sagaID, causation, operation string, data map[string]any) ([]byte, error) {
	aggregateType := "order"
	switch commandType {
	case "commerce.inventory.release.v1":
		aggregateType = "inventory"
	case "commerce.payment.authorize.v1", "commerce.payment.refund.v1":
		aggregateType = "payment"
	case "commerce.fulfillment.create.v1":
		aggregateType = "fulfillment"
	}
	return json.Marshal(map[string]any{"command_id": "cmd_" + uuid.NewString(), "command_type": commandType, "aggregate_type": aggregateType, "aggregate_id": order.ID, "aggregate_version": order.Version, "saga_id": sagaID, "correlation_id": order.ID, "causation_id": causation, "issued_at": time.Now().UTC(), "producer": "orders", "schema_version": 1, "operation_id": "ord_" + strings.TrimPrefix(order.ID, "ord_") + ":" + operationSuffix(operation), "data": data})
}
func operationSuffix(value string) string {
	if index := strings.LastIndex(value, ":"); index >= 0 {
		return value[index+1:]
	}
	return value
}
func eventPayloadFor(eventType string, order domain.Order, sagaID, causation, operation string, data map[string]any) ([]byte, error) {
	return json.Marshal(eventEnvelope{EventID: "evt_" + uuid.NewString(), EventType: eventType, AggregateType: "order", AggregateID: order.ID, AggregateVersion: order.Version, SagaID: sagaID, CorrelationID: order.ID, CausationID: causation, OccurredAt: time.Now().UTC(), Producer: "orders", SchemaVersion: 1, OperationID: operation, Data: data})
}
func rawString(value json.RawMessage) string {
	var result string
	_ = json.Unmarshal(value, &result)
	return result
}
func rawInt64(value json.RawMessage) (int64, error) {
	var result int64
	if err := json.Unmarshal(value, &result); err != nil {
		return 0, fmt.Errorf("%w: invalid number", app.ErrConflict)
	}
	return result, nil
}
func int64Null(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

type cursorValue struct {
	CustomerID string    `json:"customer_id"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	ID         string    `json:"id"`
}

func encodeCursor(value cursorValue) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func encodeSignedCursor(value cursorValue, secret []byte) (string, error) {
	encoded, err := encodeCursor(value)
	if err != nil || len(secret) == 0 {
		return encoded, err
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func decodeSignedCursor(value string, secret []byte) (cursorValue, error) {
	if len(secret) == 0 {
		return decodeCursor(value)
	}
	parts := strings.Split(value, ".")
	if len(parts) != 2 || len(parts[1]) > 128 {
		return cursorValue{}, app.ErrInvalidCursor
	}
	supplied, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return cursorValue{}, app.ErrInvalidCursor
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(supplied, mac.Sum(nil)) {
		return cursorValue{}, app.ErrInvalidCursor
	}
	return decodeCursor(parts[0])
}
func decodeCursor(value string) (cursorValue, error) {
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(data) > 1024 {
		return cursorValue{}, app.ErrInvalidCursor
	}
	var result cursorValue
	if err := json.Unmarshal(data, &result); err != nil || result.CustomerID == "" || result.ID == "" || result.CreatedAt.IsZero() {
		return cursorValue{}, app.ErrInvalidCursor
	}
	return result, nil
}

type acceptedResponse struct {
	OrderID   string    `json:"order_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
type eventEnvelope struct {
	EventID          string    `json:"event_id"`
	EventType        string    `json:"event_type"`
	AggregateType    string    `json:"aggregate_type"`
	AggregateID      string    `json:"aggregate_id"`
	AggregateVersion uint64    `json:"aggregate_version"`
	SagaID           string    `json:"saga_id"`
	CorrelationID    string    `json:"correlation_id"`
	CausationID      string    `json:"causation_id"`
	OccurredAt       time.Time `json:"occurred_at"`
	Producer         string    `json:"producer"`
	SchemaVersion    int       `json:"schema_version"`
	OperationID      string    `json:"operation_id"`
	Data             any       `json:"data"`
}

func eventPayload(eventType string, order domain.Order, sagaID, causation, operation, reason string) ([]byte, error) {
	return eventPayloadWithID("evt_"+uuid.NewString(), eventType, order, sagaID, causation, operation, reason)
}

func eventPayloadWithID(eventID, eventType string, order domain.Order, sagaID, causation, operation, reason string) ([]byte, error) {
	items := make([]map[string]any, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, map[string]any{"product_id": item.ProductID, "quantity": item.Quantity})
	}
	data := map[string]any{"order_id": order.ID, "customer_id": order.CustomerID, "currency": order.Currency, "items": items}
	if eventType == "commerce.order.cancellation_requested.v1" {
		data = map[string]any{"order_id": order.ID, "reason_code": reason}
	}
	return json.Marshal(eventEnvelope{EventID: eventID, EventType: eventType, AggregateType: "order", AggregateID: order.ID, AggregateVersion: order.Version, SagaID: sagaID, CorrelationID: order.ID, CausationID: causation, OccurredAt: order.UpdatedAt, Producer: "orders", SchemaVersion: 1, OperationID: operation, Data: data})
}

func orderItemsPayload(order domain.Order) []map[string]any {
	items := make([]map[string]any, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, map[string]any{"product_id": item.ProductID, "quantity": item.Quantity})
	}
	return items
}

func loadOrder(ctx context.Context, q *generated.Queries, id string) (domain.Order, error) {
	row, err := q.GetOrder(ctx, id)
	if err != nil {
		return domain.Order{}, mapError(err)
	}
	items, err := q.GetOrderItems(ctx, id)
	if err != nil {
		return domain.Order{}, mapError(err)
	}
	result := getOrderRowToOrder(row)
	result.Items = make([]domain.Item, 0, len(items))
	for _, item := range items {
		result.Items = append(result.Items, domain.Item{ProductID: item.ProductID, Quantity: item.Quantity, UnitPriceMinor: int64Ptr(item.UnitPriceMinor), LineTotalMinor: int64Ptr(item.LineTotalMinor), QuoteID: stringPtr(item.QuoteID), QuoteVersion: int64Ptr(item.QuoteVersion)})
	}
	return result, nil
}
func orderFromFields(id, customerID, currency string, totalMinor sql.NullInt64, reservationID, paymentID, shipmentID, failureCode sql.NullString, inventoryReleased, paymentRefunded bool, status string, version int64, createdAt, updatedAt time.Time) domain.Order {
	return domain.Order{ID: id, CustomerID: customerID, Currency: currency, TotalMinor: int64Ptr(totalMinor), ReservationID: stringPtr(reservationID), PaymentID: stringPtr(paymentID), ShipmentID: stringPtr(shipmentID), FailureCode: stringPtr(failureCode), InventoryReleased: inventoryReleased, PaymentRefunded: paymentRefunded, Status: domain.Status(status), Version: uint64(version), CreatedAt: createdAt, UpdatedAt: updatedAt}
}
func getOrderRowToOrder(row generated.GetOrderRow) domain.Order {
	return orderFromFields(row.ID, row.CustomerID, row.Currency, row.TotalMinor, row.ReservationID, row.PaymentID, row.ShipmentID, row.FailureCode, row.InventoryReleased, row.PaymentRefunded, row.Status, row.AggregateVersion, row.CreatedAt, row.UpdatedAt)
}
func cancellationRowToOrder(row generated.UpdateOrderCancellationRow) domain.Order {
	return orderFromFields(row.ID, row.CustomerID, row.Currency, row.TotalMinor, row.ReservationID, row.PaymentID, row.ShipmentID, row.FailureCode, row.InventoryReleased, row.PaymentRefunded, row.Status, row.AggregateVersion, row.CreatedAt, row.UpdatedAt)
}
func int64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}
func stringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	v := value.String
	return &v
}
func nullTime(value time.Time) sql.NullTime {
	if value.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value, Valid: true}
}
func nullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return app.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return app.ErrConflict
	}
	return fmt.Errorf("%w: %v", app.ErrDependency, err)
}
