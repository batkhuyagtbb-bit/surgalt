# surgalt.mn — «Сур, сур, бас дахин сур»

Багш нарын сургалт зарах, суралцагчид нэг дороос суралцах платформ.

Go дээр бичигдсэн, **секундэд 10 000+ хүсэлт** даах, багш бүр өөрийн нээлттэй профайл дээрээ
сургалт, хичээлээ **тус бүрчлэн (үнэгүй / төлбөртэй, үнэтэй)** зардаг систем.

## Боломжууд

| Хэсэг | Юу хийдэг |
|---|---|
| Нээлттэй профайл `/t/<нэр>` | Гэгээлэг дизайн, статистик, чиглэл, байршил, сошиал холбоос, товлосон шууд хичээл, сургалтын шүүлтүүр/хайлт, QR код |
| Ухаалаг профайл | Бүрдлийн оноо (0–100) + дараагийн алхмын зөвлөмж; нэвтэрсэн суралцагчид өөрийнх нь сургалтыг тэмдэглэнэ; эзэмшигчид засах самбар гарна |
| Суралцагчийн нүүр `/` | Нэвтэрсний дараа бүх зүйл нэг дор: сургалт, дангаар авсан хичээл, төлөгдөөгүй захиалга, шууд хичээл, багш нар, чат, "Үргэлжлүүлэх" |
| Багшийн удирдлага | Тусдаа студи байхгүй: багш өөрийн профайл дээрээ (`/t/<нэр>`) тойм, сургалт (групп шиг хичээл нийтлэх), файл, шууд хичээл, чат (хувийн + сургалтын бүлэг), тохиргоог удирдана. `/me` → өөрийн хуудас |
| Хөтөлбөр | Хичээлүүд бүлэг (модуль)-д хуваагдана: бүлгийн нэрийг шууд бичиж үүсгэнэ, хичээлийг чирж (эсвэл ↑↓ товчоор) бүлэг хооронд зөөнө (`PUT /api/courses/{id}/lesson-order`). Дараалсан нээлт (drip), хичээлийн хэлбэр (лекц, семинар, дадлага, лаборатори), заах арга (танхимын, цахим, холимог) |
| Тест, шалгалт | Асуултын 5 төрөл (нэг/олон сонголт, бичгээр, харгалзуулах, зурган дээр заах), Excel загвараар импорт (`/api/quiz-template.xlsx`). Шалгалт: хугацаа, оролдлого, тэнцэх хувь; өөр цонх руу шилжих, хуулах үед шууд хаагдана |
| Идэвхийн хяналт | Таб солиход сануулга (3 дахь удаад зогсоно), 5 минут тутам "Та үзэж байна уу?" (30 сек), хөдөлгөөнт усан тэмдэг (нэр, ID, IP), идэвхтэй суралцах хугацаа, видеог бүрэн үзэхээс өмнө урагш гүйлгэхгүй |
| Багшийн самбар | Суралцагчид таб: идэвхтэй/идэвхгүй хугацаа, анхаарлын индекс, зөрчил, шалгалт, лог, сануулга илгээх, Excel (CSV) ба PDF тайлан |
| Ном, өгүүлэл | PDF/DOC/DOCX → хуудас бүр зураг; төлөөгүй хүнд эхний N хуудас; хуудас бүрт уншигчийн тэмдэг; минутад 40 хуудасны хязгаар; бүх үйлдлийн лог |
| Идэвхтэй суралцсаны нотолгоо | Асуулт хариулах хурд (таамаг илрүүлэх), хичээлийн дараах товч дүгнэлт бичих, видеоны үзэлтийн зураглал (алгассан/давтсан хэсэг), тогтмол ирсэн өдөр; бүгд "Суралцсан оноо"-д нэгдэнэ |
| QR | `/t/<нэр>/qr.png` — уншуулахад профайл шууд нээгдэнэ, хэвлэх хэмжээгээр татна |
| Агуулга зарах | Хичээл бүр **үнэгүй** эсвэл **төлбөртэй + үнэ**; сургалтад нэмэлтээр **багц үнэ** |
| Төлбөр | Захиалга → төлбөрийн webhook (идемпотент), демо горимд `DEV_PAYMENTS=1` |
| Нэвтрэлт | Имэйл/нууц үг (давтаж баталгаажуулна), **Google (Gmail), Microsoft, Facebook, Instagram** (OAuth2/OpenID, PKCE) — түлхүүрийг `.env`-д тавихад товч гарна |
| Профайлын холбоос | Бүртгэлийн үед имэйлээс автоматаар үүснэ; багш студийн Профайл хэсгээс сольно (`PUT /api/me/username`) |
| Чат | Нэвтэрсэн **болон нэвтрээгүй** зочин багштай шууд чатлана (WebSocket) |
| Google Meet | Багш Google-ээ холбоход чатаас нэг товчоор / товлосон хичээлд Meet автоматаар үүснэ |
| Файлын сан | Багш бүрт **тусдаа хавтас** `data/teachers/<id>/{public,private}` |
| Зураг | Автоматаар **WebP**, урт тал ≤ **1000px**, EXIF эргэлт засна, GPS мэдээлэл арилна |
| Видео | Ард **WebM (VP9+Opus)** болгоно (ffmpeg) |
| Баримт | Word/PowerPoint/Excel → PDF (LibreOffice); бүх PDF **3D ном** шиг эргүүлж уншина |
| Багтаамжийн төлбөр | Үнэгүй **300MB** + нэмэлт **300MB / 500MB / 1GB / 2GB / 5GB** сарын багц |
| Мэдэгдэл | Үзэлт, худалдан авалт, элсэлт, мессеж → бодит цагийн мэдэгдэл + браузерийн notification |
| Responsive | Утас (≤560px), таблет (561–1100px), компьютер — бүгд шалгагдсан |

## Өгөгдлийн сан: ClickHouse (бүх өгөгдөл)

Хэрэглэгч, сургалт, хичээл, захиалга, чат, мэдэгдэл, шалгалт, идэвхийн лог, ном — бүгд ClickHouse-д.
`CLICKHOUSE_DSN` хоосон бол санах ойн store (хөгжүүлэлт, өгөгдөл хадгалагдахгүй).

```sh
# Локал (Docker-гүй): нэг binary
sh deploy/clickhouse-local.sh                      # :9000 native, :8123 http
CLICKHOUSE_DSN=clickhouse://127.0.0.1:9000/surgalt ./surgalt
# Docker: docker compose up -d --build  (Keeper + ClickHouse + app ×3 + nginx)
```

ClickHouse нь OLAP сан тул store дараах зарчмаар бичигдсэн (`internal/store/clickhouse*.go`):

- Өөрчлөгддөг entity бүр `ReplacingMergeTree(ver)`: засвар = шинэ хувилбартай мөр, уншихдаа `FINAL`; устгал = `deleted` тэмдэг.
- Зөвхөн нэмэгддэг өгөгдөл (мессеж, лог, үйл явдал) — `MergeTree`; тоолуур — `SummingMergeTree`.
- **Давхардал ба "яг нэг удаа"** (имэйл/нэрний давхардал, төлбөр, pending захиалга) — `Locker` (`internal/store/locker.go`):
  нэг хуулбарт процесс доторх мутекс; олон хуулбарт **ClickHouse Keeper** (ZooKeeper-compatible, ClickHouse-ийн нэг хэсэг)
  дээрх түгжээ (`CLICKHOUSE_KEEPER=host:9181`). Имэйлд нэмээд бичсэний дараах шалгалт бий: ижил имэйлтэй хэд хэдэн
  мөр үүсвэл хамгийн эрт ID нь үлдэж, бусад нь `409` авна — түгжээ алдагдсан ч давхардахгүй.
- **Олон хуулбарт чат, мэдэгдэл**: Keeper тохируулсан үед хуулбар бүр ClickHouse-оос шинэ мөрүүдийг (300мс) санамж авч
  өөрийн WebSocket-ууд руу хүргэнэ — ClickHouse өөрөө "bus", нэмэлт брокер хэрэггүй.
- Тест: `SURGALT_TEST_CLICKHOUSE_DSN=clickhouse://127.0.0.1:9000/default [SURGALT_TEST_KEEPER=127.0.0.1:9181] go test ./internal/httpapi`
  — HTTP API-ийн бүх тест жинхэнэ ClickHouse (+Keeper түгжээ) дээр ажиллана; `TestRegisterDuplicateEmailConcurrent`
  нэг имэйлээр 20 зэрэг бүртгэлээс яг 1 амжилттай гарахыг шалгана.

## Protobuf API: gRPC + gRPC-Web + Connect-JSON (нэг handler)

Схем: `proto/surgalt/v1/surgalt.proto` → `internal/pb` (+ `internal/pb/pbconnect`), үйлчилгээ: `internal/grpcapi`.
[Connect](https://connectrpc.com) handler нэг замаар (`/surgalt.v1.Surgalt/<Method>`) гурван протоколыг үйлчилнэ:

| Клиент | Протокол | Хэрхэн |
|---|---|---|
| Мобайл, сервис (grpc-go, Swift, Kotlin …) | gRPC (HTTP/2, h2c) | `:8080` эсвэл `GRPC_ADDR` (`:9090`, nginx `grpc_pass`) |
| Хөтөч (gRPC-Web клиент) | gRPC-Web | ижил зам |
| Хөтөч (энгийн `fetch`) | Connect-JSON | `POST /surgalt.v1.Surgalt/Search`, `Content-Type: application/json` |

Вэб хэсэг өөрөө энэ API-г ашигладаг: нүүрийн хайлт (`Search`), хичээлийн идэвхийн сесс (`StartSession`, `BeatOnce`).
Токен: `Authorization: Bearer <token>`. HTTP ба Protobuf API нэг л бизнес логик (`httpapi/service.go`), нэг л ClickHouse.

| RPC | Хэн | Юу |
|---|---|---|
| `GetTeacher`, `GetCourse`, `Search` | нээлттэй | Профайл, сургалт + хичээлийн тойм, ухаалаг хайлт |
| `Me` | нэвтэрсэн | Өөрийн мэдээлэл |
| `StartSession`, `Beat` (bidi stream), `BeatOnce` | нэвтэрсэн | Идэвхийн сесс, 15 сек тутмын тайлан → ClickHouse |
| `GetAnalytics` | багш | Суралцагч бүрийн идэвх, анхаарал, суралцсан оноо |

Код дахин үүсгэх (`protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-connect-go` нь `go install`-аар):
`protoc --proto_path=proto --go_out=internal/pb --go_opt=module=surgalt/internal/pb --go-grpc_out=internal/pb --go-grpc_opt=module=surgalt/internal/pb --connect-go_out=internal/pb --connect-go_opt=module=surgalt/internal/pb proto/surgalt/v1/surgalt.proto`

## Архитектур

```
nginx (load balancer, WebSocket, том upload урсгал)
   │
   ├── app ×3 (Go) ──── ClickHouse (бүх өгөгдөл) + ClickHouse Keeper (хуулбар хоорондын түгжээ)
   │     ├─ Protobuf API (Connect): gRPC / gRPC-Web / JSON — хөтөч, мобайл, сервис нэг зам
   │     ├─ процесс доторх кэш + singleflight (stampede хамгаалалт)
   │     ├─ урьдчилан рендерлэсэн HTML/JSON + ETag/304
   │     ├─ IP тус бүрийн rate limit, bcrypt-ийн зэрэгцээг хязгаарлана
   │     ├─ үзэлтийг санах ойд нэгтгэж 30с тутам багцаар бичнэ
   │     └─ чат/мэдэгдэл: WebSocket hub; олон хуулбарт ClickHouse-оор дамжуулан түгээнэ
   └── файлын сан (хуваалцсан volume)
```

```
cmd/server        — эхлэл, тохиргоо, демо өгөгдөл
cmd/loadtest      — open-model ачааллын тест
internal/store    — Store интерфэйс; ClickHouse (ReplacingMergeTree + FINAL) ба санах ойн (dev/тест) хэрэгжүүлэлт
internal/grpcapi  — gRPC + Protobuf үйлчилгээ (proto/surgalt/v1)
internal/httpapi  — REST API, WebSocket, HTML загвар, статик (embed)
internal/cache    — shard-лагдсан TTL кэш + singleflight
internal/files    — багшийн файлын сан, WebP, видео/баримт хөрвүүлэлт, гарын үсэгтэй URL
internal/chat     — WebSocket hub
internal/auth     — HMAC токен, bcrypt, AES-GCM
internal/oauth    — Google, Microsoft, Facebook, Instagram
internal/meet     — Google Calendar/Meet
internal/ratelimit
```

## Ачааллын тестийн үр дүн

8 цөмтэй нэг машин дээр (клиент ба сервер хамт), ClickHouse-тэй, 1 сервер:

| Ачаалал | Амжилттай | p50 | p99 | Серверийн санах ой |
|---|---|---|---|---|
| 10 000 req/s × 10с | 100 000 / 100 000 | 0.28мс | 1.2мс | 33MB |
| 30 000 req/s × 10с | 300 000 / 300 000 | 0.34мс | 0.9мс | 40MB |

Холимог замууд: профайл HTML/JSON, сургалт HTML/JSON, үнэгүй хичээл.

```bash
go run ./cmd/loadtest -base http://localhost:8080 -rate 10000 -d 10s -paths /t/demo,/api/teachers/demo
```

> Тест нэг IP-ээс явуулдаг тул `RATE_LIMIT_RPS=0` тавьж ажиллуулна.

## Ажиллуулах

**Хурдан (өгөгдлийн сангүй, санах ойд):**
```bash
SEED_DEMO=1 DEV_PAYMENTS=1 go run ./cmd/server
# http://localhost:8080/t/demo   — демо багш (demo@surgalt.mn / demo12345)
# http://localhost:8080/          — демо суралцагч (suragch@surgalt.mn / demo12345)
```

**Production (Docker):**
```bash
cp .env.example .env     # TOKEN_SECRET, PUBLIC_URL, OAuth түлхүүрүүдийг бөглөнө
docker compose up -d --build
```

## Гадны үйлчилгээ тохируулах

- **Google / Microsoft / Facebook / Instagram:** developer console-д app үүсгээд
  callback-ийг `{PUBLIC_URL}/auth/<google|microsoft|facebook|instagram>/callback` гэж бүртгэнэ.
  Instagram нь зөвхөн мэргэжлийн (business/creator) дансаар нэвтэрдэг бөгөөд имэйл өгдөггүй.
- **Google Meet:** Google Cloud-д Calendar API идэвхжүүлж, нэмэлт callback
  `{PUBLIC_URL}/auth/google-meet/callback` бүртгэнэ.
- **Төлбөр (QPay г.м):** `handleEnroll`, `handleBuyLesson`, `handleBuyStorage` дээр invoice үүсгэж,
  төлөгдсөний дараа үйлчилгээ `POST /api/payments/webhook` (`X-Webhook-Secret` толгойтой)-г дуудна.

## Тест

```bash
go test -race ./...
```
