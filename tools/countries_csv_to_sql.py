import csv
import sys
import os

"""
This script converts a CSV file of countries into SQL INSERT statements.

Your countries.csv should look like this:
Alpha-3 Code,Name (English),Name (German),Name (Danish)
USA,United States,Vereinigte Staaten,USA
DEU,Germany,Deutschland,Tyskland

Usage:
    python3 csv_to_sql.py countries.csv
"""
def csv_to_sql(input_csv):
    output_sql = os.path.splitext(input_csv)[0] + '.sql'

    with open(input_csv, mode='r', encoding='utf-8') as infile, open(output_sql, mode='w', encoding='utf-8') as outfile:
        reader = csv.DictReader(infile)
        inserts = []

        for row in reader:
            code = row['Alpha-3 Code'].strip()
            name_en = row['Name (English)'].strip().replace("'", "''")
            name_de = row['Name (German)'].strip().replace("'", "''")
            name_da = row['Name (Danish)'].strip().replace("'", "''")

            inserts.append(f"('{code}', '{name_en}', 'en')")
            inserts.append(f"('{code}', '{name_de}', 'de')")
            inserts.append(f"('{code}', '{name_da}', 'da')")

        outfile.write("INSERT INTO countries (code, name, iso_639_1) VALUES\n")
        outfile.write(",\n".join(inserts))
        outfile.write(";\n")

    print(f"SQL INSERT statements written to {output_sql}")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 csv_to_sql.py <inputfile.csv>")
        sys.exit(1)

    input_csv = sys.argv[1]
    csv_to_sql(input_csv)