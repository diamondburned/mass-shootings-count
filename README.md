# mass-shootings-count

[shootings.libdb.so](https://shootings.libdb.so)

Website that shows the number of days since the last mass shooting in the United
States. It is made to raise awareness.

## Usage

```sh
go run ./cmd/mass-shootings-count
go run ./cmd/mass-shootings-count -addr localhost:8081
```

## API

```sh
curl -H 'Accept: application/json' https://shootings.diamondb.xyz/ | jq .
```

```json
{
  "Days": 0,
  "Records": [
    {
      "IncidentID": 2349853,
      "IncidentDate": "July 5, 2022 EDT",
      "State": "Indiana",
      "CityCounty": "Gary",
      "Address": "1900 block of Missouri St",
      "NoKilled": 3,
      "NoInjured": 7,
      "IncidentURL": "https://gunviolencearchive.org/incident/2349853",
      "SourceURL": "https://www.cbsnews.com/chicago/news/3-dead-7-wounded-after-shooting-at-block-party-in-gary/"
    }
  ],
  "LastUpdated": "2022-07-05T15:27:57.995562655-07:00"
}
```
