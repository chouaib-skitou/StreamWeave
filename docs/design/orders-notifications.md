# Orders Notification Boundary

## Ownership

Orders owns order truth and publishes versioned order facts. Notifications owns templates, recipient resolution, provider credentials, delivery retries, suppression, and delivery status. Orders never calls an email provider and never waits for notification delivery.

## Subscription contract

Notifications consumes `commerce.order.events.v1` with its own consumer group and Inbox. It may subscribe to `commerce.order.created.v1`, `commerce.order.confirmed.v1`, `commerce.order.cancelled.v1`, `commerce.order.failed.v1`, and `commerce.order.completed.v1` according to the notification policy. Each fact already contains the order ID, customer reference, status-specific reason where safe, correlation ID, and operation ID.

The customer reference is resolved through an approved Identity-owned profile boundary; Orders does not publish email addresses, access tokens, payment data, or address snapshots into notification events. A missing recipient or provider outage is a Notifications delivery failure, not an Orders Saga failure.

## Verification scenarios

- A duplicate order fact creates at most one notification intent per `(consumer_group, event_id, notification_type)`.
- A failed provider attempt is retried by Notifications and does not change the Orders status.
- A terminal order fact remains queryable even when Notifications is stopped.
- Notification logs and metrics contain provider/result classes but no address, token, or message body.
