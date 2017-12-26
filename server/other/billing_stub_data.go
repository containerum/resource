package other

import (
	"encoding/json"
)

var fakeNSData = `
[
  {
    "id": "3c9d98af-ef8c-4486-ba28-01d83bdd2ddd",
    "tariff_id": "f3091cc9-6dc3-470e-ac54-84defe011111",
    "created_at": "2017-12-26T13:53:56Z",
    "cpu_limit": 500,
    "memory_limit": 512,
    "traffic": 20,
    "traffic_price": 0.333,
    "external_services": 2,
    "internal_services": 5,
    "VV": "15348470-e98f-4da0-8d2e-8c65e15d6eeb",
    "is_active": true,
    "is_public": true,
    "price": 0
  },
  {
    "id": "2f7f294d-3f53-4b10-94e2-e7411570d9a7",
    "tariff_id": "4563e8c1-fb41-416a-9798-e949a2616260",
    "created_at": "2017-12-26T13:57:45Z",
    "cpu_limit": 900,
    "memory_limit": 1024,
    "traffic": 50,
    "traffic_price": 0.5,
    "external_services": 10,
    "internal_services": 20,
    "VV": "11a35f90-c343-4fc1-a966-381f75568036",
    "is_active": true,
    "is_public": true,
    "price": 0
  }
]
`

var fakeVolumeData = `
[
]
`

var fakeNSTariffs []NamespaceTariff
var fakeVolumeTariffs []VolumeTariff
