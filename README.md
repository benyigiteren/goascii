# go-ascii

Terminal üzerinden akan ASCII animasyon servisi. Go ile yazıldı,
bağımlılığı yok, tek binary olarak çalışır.

Bir web sunucusu olarak 8080 portunda dinler. Tarayıcıdan girerseniz
yönetim paneline yönlendirir. `curl` veya `wget` ile girerseniz doğrudan
animasyonun canlı akışını yazdırır.

## Kurulum

```bash
go build -o go-ascii.exe .
./go-ascii.exe
```

`PORT` ortam değişkeni ile farklı port seçilebilir.

## Kullanım

```bash
# Tarayıcıdan JSON yardım listesi
curl https://localhost:8080/

# Bir animasyonu canlı izle (Ctrl+C ile çık)
curl -N https://localhost:8080/nyancat
```

### Sorgu parametreleri

- `w` — Genişlik (varsayılan: 80)
- `h` — Yükseklik (varsayılan: 24)
- `color=false` — Renkleri kapatır

```bash
curl -N "https://localhost:8080/matrix?w=120&h=40&color=false"
```

## Animasyonlar

### Matematiksel / 3D (7)

- `earth` — Dönen 3D dünya küresi, kıtasal gölgelendirme
- `matrix` — Matrix tarzı dijital yağmur
- `donut` — Dönen 3D donut (z-buffer + ışıklandırma)
- `cube` — Dönen 3D tel kafes küp
- `fire` — Doom tarzı prosedürel alev simülasyonu
- `nyancat` — Gökkuşağı izli pop-tart kedi
- `crewmate` — Among Us crewmate, yürüyen ve renk değiştiren

### Hacker (3)

- `hacker1` — Yukarı kayan sahte kaynak kodu logları
- `hacker2` — Canlı terminal, harf harf yazılan komutlar + prompt
- `hacker3` — Binary / sembol yağmuru, glitch efektleri

### Meme (5)

- `doge` — Doge yüzü, göz kırpma, dönen "wow / such ascii" yazıları
- `dancer` — Dans eden stick figure + never gonna give you up
- `dab` — Dab yapan stick figure
- `rock` — Rock on eli, parıltılar
- `troll` — Trollface zoom + "PROBLEM?" yazısı

## Yönetim Paneli

Tarayıcıdan `https://localhost:8080/` adresine girdiğinizde otomatik
olarak yönetim paneline yönlendirilirsiniz. Buradan:

- Kayıt olup giriş yapabilirsiniz
- GIF veya MP4 yükleyip özel animasyonlar olarak ekleyebilirsiniz
- Animasyonları silebilirsiniz
- Erişim loglarını görebilirsiniz

Tüm veriler `./data/db.json` üzerinde, yüklenen animasyonlar
`./data/animations/` altında tutulur.

## Mimari

```
main.go                — Router, server bootstrap
pkg/ascii/             — Prosedürel frame üreteçleri
pkg/db/                — JSON tabanlı kalıcı veri katmanı
pkg/admin/             — Yönetim paneli HTTP handler'ları
web/templates/         — HTML şablonları (embed)
web/static/            — Statik dosyalar (embed)
```

Tüm HTTP chunked stream üzerinden akar, frame başına ANSI kaçış
kodları ile terminal temizleme yapılır.

## Docker ile Çalıştırma

Repository'de multi-stage `Dockerfile`, `docker-compose.yml` ve
opsiyonel `Caddyfile` bulunur. Caddy, Let's Encrypt sertifikasını
otomatik alır **ve** hem `http://` hem `https://` üzerinden aynı anda
serve eder; `http -> https` yönlendirmesi yapmaz.

### Yalnızca Go (Dokploy / Traefik ile)

```bash
docker build -t go-ascii .
docker run -d --name go-ascii -p 8080:8080 -v goascii_data:/app/data go-ascii
```

Dokploy üzerinde `docker-compose.yml` veya Dockerfile olarak deploy
edersen, Traefik otomatik `https://ascii.yigiteren.org` üzerinden
yayınlar. `http://` üzerinden de aynı içerik gelir.

### Go + Caddy (Traefik'siz, otomatik TLS)

`docker-compose.yml` içindeki `caddy` servisinin başındaki yorum
satırlarını kaldır ve domain'i değiştir:

```yaml
caddy:
  image: caddy:2-alpine
  ports: ["80:80", "443:443"]
  volumes:
    - ./Caddyfile:/etc/caddy/Caddyfile:ro
    - caddy_data:/data
    - caddy_config:/config
```

Sonra:

```bash
docker compose up -d
```

Bu kurulumda:
- `curl https://ascii.yigiteren.org/earth` çalışır (TLS, Let's Encrypt)
- `curl http://ascii.yigiteren.org/earth` da çalışır (TLS yok, redirect yok)
- `Moved Permanently` almazsın

### Ortam Değişkenleri

| Değişken         | Varsayılan | Açıklama                                               |
|------------------|------------|--------------------------------------------------------|
| `PORT`           | `8080`     | Ana HTTP portu                                         |
| `HTTP_PORT_2`    | boş        | İkincil HTTP portu (örn. `80` host'a bağlamak için)   |
| `HTTPS_PORT`     | boş        | TLS portu (örn. `443`), sadece sertifika varsa dinler  |
| `TLS_CERT_FILE`  | boş        | `fullchain.pem` yolu                                   |
| `TLS_KEY_FILE`   | boş        | `privkey.pem` yolu                                     |

Sunucu **asla** `Location` header'ı ile yönlendirme yapmaz. Caddy
veya başka reverse proxy ile zorunlu https istersen onlar ekler;
istemezsen hiç eklenmez.
