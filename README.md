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
