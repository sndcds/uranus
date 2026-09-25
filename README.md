# Uranus

**Uranus** is a flexible, open-source event management platform designed to organize, schedule, and promote cultural activities. It empowers non-profits and grassroots organizations to manage events, venues, and organizers in a structured way, providing accessible, high-quality data via a robust API.

# Key Features

- **Structured Event Data** – Organize events, venues, and spaces with detailed metadata, including accessibility info and geospatial data for precise location mapping and spatial queries.
- **Flexible API** – Access and integrate event data via standardized HTTP endpoints.
- **Support for Nonprofits** – Tailored for smaller cultural and civic organizations that need digital infrastructure.
- **Inclusive Values** – Supports diversity, minority representation, gender equality, and sustainability.
- **Open Source** – Reuse, modify, and redistribute freely under a permissive license.


# Building Blocks

- **Uranus API Server** – Go-based backend for fast and scalable data access.
- **PostgreSQL + PostGIS** – Database with spatial support for accurate venue and event mapping.
- **Pluto Image Server** – Integrated image handling for events, venues, spaces and organizations.


# Getting Started

1. Clone the repository for the Vue3 frontend:
   ```bash
   git clone git@github.com:sndcds/uranus-dashboard.git
   ```

2. Clone the repository for the Go backend:
	```bash
	git clone clone git@github.com:sndcds/uranus.git
   ```

3.	Follow setup instructions in the documentation to run the API server and database.


# API Documentation

The combined OpenAPI 3.0.3 specification is available in
[`openapi/oas3-combined.yaml`](openapi/oas3-combined.yaml). It includes all
22 endpoints documented in the individual OpenAPI files and preserves their
response definitions and examples, including endpoint-specific error responses.
Schemas with repeated names have endpoint-specific prefixes to keep their
original definitions distinct. The individual specifications remain available
in `openapi/`.

A German [developer overview of event endpoints and lookups](docs/event-endpoints-und-lookups.md)
maps event fields to their lookup endpoints and describes filters, pagination,
and known documentation gaps.

[Social rendering and preview (Part 4)](docs/social-preview.md) documents the
admin preview endpoint, platform policies, architecture and tests.


# Contributing

We welcome contributions, feedback, and feature requests! You can:

- Report bugs
- Suggest new features
- Fork and extend the code for your own needs

Join the community and help us build a better platform for cultural and civic event management.
