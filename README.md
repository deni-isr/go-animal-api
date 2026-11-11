# Go Animal REST API

Go-kielellä (Gin + GORM) ja MariaDB-tietokannalla toteutettu REST API.

## Suoritetut vaiheet (Completed Steps)

- **Step 1 (Tutorials):** Tutustuttu Go REST API -tutoriaaleihin.
- **Step 2 (CRUD + MariaDB):** Toteutettu täydet CRUD-operaatiot (`GET`, `POST`, `PUT`, `DELETE`) MariaDB-tietokannalla. Tietokanta ja testidata luodaan automaattisesti käynnistyksessä.
- **Step 3 (Advanced features):** Toteutettu 4 ominaisuutta:
  1. **Filtering:** `?species=Cat`, `?habitat=Savannah`, `?endangered=true`, `?min_age=3`
  2. **Sorting:** `?sort=age&order=desc`
  3. **Pagination:** `?page=1&limit=5`
  4. **Validation:** Syötteiden tarkistus `POST`- ja `PUT`-pyynnöissä (`required`, `min`, `max`, `gte`, `lte`).
- **Step 4 (Documentation):** OpenAPI 3.0 (`openapi.yaml`) ja Swagger UI osoitteessa `/swagger`.
- **Step 5 (Docker):** Kontitettu `Dockerfile`- ja `docker-compose.yml`-tiedostoilla.

## Käynnistys

### Paikallisesti (Go + MariaDB/XAMPP)

go mod tidy
go run main.go


## Dockerilla

docker compose up --build

## Reitit (Endpoints)

Swagger UI: http://localhost:8080/swagger

GET /api/v1/animals – Kaikki eläimet (tukee suodatusta, järjestystä ja sivutusta)

GET /api/v1/animals/:id – Yksittäinen eläin

POST /api/v1/animals – Lisää uusi eläin

PUT /api/v1/animals/:id – Päivitä eläin

DELETE /api/v1/animals/:id – Poista eläin
