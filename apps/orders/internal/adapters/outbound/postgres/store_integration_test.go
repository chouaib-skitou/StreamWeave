//go:build integration

package postgres

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chouaib-skitou/streamweave/apps/orders/internal/adapters/outbound/postgres/generated"
	app "github.com/chouaib-skitou/streamweave/apps/orders/internal/application/orders"
	domain "github.com/chouaib-skitou/streamweave/apps/orders/internal/domain/orders"
	"github.com/google/uuid"
)

func TestCancellationRejectionCommitsOrderAndInbox(t *testing.T) {
	// Use a dedicated test database: this test applies the real migrations.
	databaseURL := os.Getenv("ORDERS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("ORDERS_TEST_DATABASE_URL must point to a dedicated test database")
	}
	connection, err := Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err = connection.Migrate(ctx, filepath.Join("..", "..", "..", "..", "..", "..", "database", "orders", "migrations")); err != nil {
		t.Fatal(err)
	}
	store := NewStore(connection)
	orderID := "ord_" + uuid.NewString()[:8]
	actorID := "usr_review"
	order, err := domain.NewOrder(orderID, actorID, "EUR", []domain.Item{{ProductID: "prd_review", Quantity: 1}}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Create(ctx, actorID, order, app.CreateOperation, "create-"+orderID, "create-hash"); err != nil {
		t.Fatal(err)
	}
	if _, err = store.RequestCancellation(ctx, actorID, orderID, app.CancelOperation, "cancel-"+orderID, "cancel-hash", "CUSTOMER_REQUESTED", order.Version, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	saga, err := store.q.GetSagaByOrderID(ctx, orderID)
	if err != nil {
		t.Fatal(err)
	}
	eventID := "evt_" + uuid.NewString()
	fact := map[string]any{"event_id": eventID, "event_type": "commerce.inventory.rejected.v1", "aggregate_id": "inv_review", "aggregate_type": "inventory", "producer": "inventory", "schema_version": 1, "saga_id": saga.ID, "correlation_id": orderID, "operation_id": orderID + ":reserve", "data": map[string]any{"order_id": orderID, "reason_code": "OUT_OF_STOCK"}}
	payload, err := json.Marshal(fact)
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if err = store.ProcessFact(ctx, "orders-review", payload); err != nil {
			t.Fatal(err)
		}
		persisted, err := store.FindByID(ctx, orderID)
		if err != nil {
			t.Fatal(err)
		}
		if persisted.Status != domain.StatusCancelled || !persisted.InventoryReleased {
			t.Fatalf("compensation did not persist: %+v", persisted)
		}
		inbox, err := store.q.GetInbox(ctx, generated.GetInboxParams{ConsumerGroup: "orders-review", MessageID: eventID})
		if err != nil {
			t.Fatal(err)
		}
		if inbox.Status != "PROCESSED" {
			t.Fatalf("Inbox was not committed: %s", inbox.Status)
		}
	}
}
