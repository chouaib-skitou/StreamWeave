# Staging monitoring

Staging uses one shared monitoring release for every StreamWeave service. Do
not deploy a Prometheus, Alertmanager, Grafana, Tempo, or collector instance
inside an individual service chart.

The staging release must run in a dedicated monitoring namespace and use:

- external secret management for Grafana, alerting, and notification credentials;
- TLS and authenticated access for Grafana, Prometheus, Alertmanager, and OTLP;
- durable storage with staging retention and resource limits;
- service discovery for every service `/metrics` endpoint;
- alert routing to the staging channel with non-production paging policy.

The service Helm charts expose scrape configuration only; platform operators
own this shared release.
