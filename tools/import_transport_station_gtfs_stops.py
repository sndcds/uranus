#!/usr/bin/env python3
import csv
import psycopg2
from psycopg2.extras import execute_values
from typing import Optional
import argparse

'''
Usage:
    python3 tools/import_stations.py \
        --csv stops.csv \
        --city Flensburg \
        --country DEU \
        --host localhost \
        --port 5432 \
        --db uranus \
        --user postgres \
        --password mysecretpassword

    python3 tools/import_transport_station_gtfs_stops.py --csv /Volumes/RoaldMedia1/OpenData/Transport/GTFS/SH/stops.txt --city Flensburg --country DEU --host localhost --port 5432 --db oklab --user roaldchristesen --password ''
'''

# --- your import function ---
def import_transport_stations(
    csv_file_path: str,
    city: Optional[str] = None,
    country: Optional[str] = None,
    db_config: dict = None
):
    if db_config is None:
        db_config = {
            "host": "localhost",
            "database": "uranus",
            "user": "postgres",
            "password": "",
        }

    conn = psycopg2.connect(**db_config)
    cur = conn.cursor()

    insert_sql = """
    INSERT INTO uranus.transport_station (
        name, geo_pos, gtfs_station_code, gtfs_location_type,
        city, country, gtfs_parent_station, gtfs_wheelchair_boarding, gtfs_zone_id
    ) VALUES %s
    ON CONFLICT (gtfs_station_code) DO UPDATE
    SET
        name = EXCLUDED.name,
        geo_pos = EXCLUDED.geo_pos,
        gtfs_location_type = EXCLUDED.gtfs_location_type,
        city = EXCLUDED.city,
        country = EXCLUDED.country,
        gtfs_parent_station = EXCLUDED.gtfs_parent_station,
        gtfs_wheelchair_boarding = EXCLUDED.gtfs_wheelchair_boarding,
        gtfs_zone_id = EXCLUDED.gtfs_zone_id
    """

    values = []

    with open(csv_file_path, newline="", encoding="utf-8") as csvfile:
        reader = csv.DictReader(csvfile)
        for row in reader:
            stop_name = row.get("stop_name", "").strip()
            stop_code = row.get("stop_id", "").strip() or None
            lat = row.get("stop_lat")
            lon = row.get("stop_lon")
            location_type = int(row.get("location_type", 0))
            parent_station = row.get("parent_station") or None
            wheelchair_boarding = int(row.get("wheelchair_boarding", 0))
            zone_id = row.get("zone_id") or None

            try:
                lat_f = float(lat)
                lon_f = float(lon)
            except (TypeError, ValueError):
                print(f"Skipping invalid row: {row}")
                continue

            point_wkt = f"SRID=4326;POINT({lon_f} {lat_f})"

            values.append((
                stop_name,
                point_wkt,
                stop_code,
                location_type,
                city,
                country,
                parent_station,
                wheelchair_boarding,
                zone_id
            ))

    if not values:
        print("No valid rows to insert.")
        cur.close()
        conn.close()
        return

    execute_values(
        cur,
        insert_sql,
        values,
        template="""(
            %s,
            ST_GeomFromText(%s),
            %s,
            %s,
            %s,
            %s,
            %s,
            %s,
            %s
        )"""
    )

    conn.commit()
    cur.close()
    conn.close()
    print(f"Imported {len(values)} transport stations successfully.")


# --- CLI entry point ---
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Import GTFS stops into transport_station table")
    parser.add_argument("--csv", required=True, help="Path to the CSV file")
    parser.add_argument("--city", default=None, help="City name to assign")
    parser.add_argument("--country", default=None, help="Country code to assign")
    parser.add_argument("--host", default="localhost", help="PostgreSQL host")
    parser.add_argument("--port", default=5432, type=int, help="PostgreSQL port")
    parser.add_argument("--db", required=True, help="Database name")
    parser.add_argument("--user", required=True, help="Database user")
    parser.add_argument("--password", required=True, help="Database password")

    args = parser.parse_args()

    db_config = {
        "host": args.host,
        "port": args.port,
        "database": args.db,
        "user": args.user,
        "password": args.password,
    }

    import_transport_stations(args.csv, args.city, args.country, db_config)