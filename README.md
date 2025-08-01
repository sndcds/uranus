# Uranus

**Digitally Mapping and Sharing the Event Landscape**

> 🚧 Disclaimer: This repository and the associated database are currently in **beta**. Some parts of the code and data may still contain errors. Please [open an issue](#) or contact us via email if you encounter any problems.

---

## Introduction

**Uranus** aims to create a detailed and flexible representation of the event landscape, making it easier to create, maintain, publish, and share high-quality event data.

The database models:
- **Venues**: Physical locations of events
- **Spaces**: Descriptions of rooms or sub-locations within venues
- **Organizations**: Represented hierarchically (e.g., Institution → Association → Working Group)
- **Events**: Including flexible, multi-date scheduling via `EventDate`

The data is accessible through an **open API** for use in:
- Plugins (e.g., WordPress)
- Website integrations
- Custom applications

Unlike other systems that focus solely on single events, Uranus emphasizes the **relationships** between locations, dates, and organizers — enabling advanced search and visualizations (e.g., maps and portals).

---

## Target Audiences

- **Event Organizers**: Associations, initiatives, educational and cultural institutions, etc.
- **Event Enthusiasts**: Users searching for events based on:
	- Date
	- Audience
	- Type/genre
	- Location
	- Organizers
	- Accessibility
- **Portals & Institutions**: Municipalities, cultural associations, or media platforms that want to integrate event data
- **Journalists & Culture Reporters**: Looking for structured event information

---

## Project Status

This project was initiated by **OK Lab Flensburg** on **March 2, 2025**.

We are currently building the **first MVP** (Minimal Viable Product) to demonstrate the core concept of Uranus. The MVP will include:
- Event creation & management
- API access

More features will be added in the coming months.

---

## Get Involved

We welcome feedback and contributions.  
Feel free to:
- Create issues for bugs or feature ideas
- Submit pull requests
- Contact us directly via email

---

## Installation

---

### Prerequisites

1. **Database Setup**

- Ensure PostgreSQL with PostGIS extension is installed and running on `localhost` (default port: `5432`).
- Create a database named `uranus`, owned by a user with the same name.
- Make sure the database accepts connections from `localhost`.

2. **Environment Variables**


---

## Configuration

```json
{
  "verbose": true,
  "dev_mode": true,
  "port": 9090,
  "base_api_url": "http://localhost:9090",
  "use_router_middleware": true,
  "db_host": "localhost",
  "db_port": 5432,
  "db_user": "roaldchristesen",
  "db_password": "",
  "db_name": "oklab",
  "db_schema": "uranus",
  "ssl_mode": "disabled",
  "allow_origins": ["http://localhost:8009", "https://uranus.grain.one"],
  "pluto_verbose": true,
  "pluto_image_dir": "/Users/roaldchristesen/Documents/Developer/Projects/pluto/image",
  "pluto_cache_dir": "/Users/roaldchristesen/Documents/Developer/Projects/pluto/cache",
  "jwt_secret": "82jhdksl#",
  "auth_token_expiration_time": 3600
}
```


---

## Export Database

```sh
pg_dump -U oklab -h localhost -d oklab -n app --data-only --column-inserts --no-owner --no-comments --verbose -f uranus_data_dump.sql
pg_dump -U oklab -h localhost -d oklab -n app --schema-only --no-owner --no-comments --verbose -f uranus_schema_dump.sql
```
