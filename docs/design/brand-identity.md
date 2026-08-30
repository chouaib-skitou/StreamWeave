# Event Commerce Platform Brand Identity

## Direction

Event Commerce Platform uses a confident dark foundation with a bright operational accent. The visual language combines dependable event flows with the energy of commerce: deep midnight surfaces, electric cyan actions, and a restrained violet signal for secondary emphasis.

## Palette

| Token | Value | Use |
|---|---|---|
| Midnight | `#0B1020` | Email headers, navigation, primary dark surfaces |
| Ink | `#0F172A` | Headlines and high-contrast text |
| Electric Cyan | `#06B6D4` | Primary actions, links, active states |
| Cyan Mist | `#9EEEFF` | Accent text on dark surfaces |
| Violet Signal | `#7C3AED` | Secondary accent and event-flow highlights |
| Cloud | `#F5F7FB` | Email canvas and application backgrounds |

## Logo

The platform mark combines a shopping bag with an event pulse/orbit. The canonical generated raster asset is embedded in Identity transactional emails as an inline CID image:

- `apps/identity/internal/platform/runtime/assets/platform-logo.png`

The asset has no credentials, user data, or environment-specific content. Keep the logo clear of additional badges, overlays, or text when reusing it in future services.

## Email system

All transactional Identity emails share one shell: brand header, greeting, variable content, single action button, security note, support contact, and legal footer. Only the subject and body copy vary by event. The HTML and plain-text alternatives must remain semantically equivalent.
