```bash
ICS_URL='https://www.uni-flensburg.de/kommunikation/veranstaltungen-und-termine/events/termine.ics' \
go test -v -run '^TestReadICSURL$' ./service
```