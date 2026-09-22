# Panduan Penggunaan AnyFlip Downloader

Panduan lengkap untuk mengunduh buku digital dari AnyFlip dan mengonversinya menjadi dokumen PDF berkualitas tinggi, baik melalui **Antarmuka Grafis (UI Desktop)** maupun **Baris Perintah (CLI)**.

---

## 🌟 Daftar Isi
1. [Cara Menjalankan Mode UI Desktop](#1-cara-menjalankan-mode-ui-desktop)
2. [Fitur-Fitur pada Antarmuka (UI)](#2-fitur-fitur-pada-antarmuka-ui)
3. [Cara Menjalankan Mode CLI (Terminal)](#3-cara-menjalankan-mode-cli-terminal)
4. [Tabel Parameter & Flag CLI](#4-tabel-parameter--flag-cli)
5. [Kompilasi / Build Ulang dari Kode Sumber](#5-kompilasi--build-ulang-dari-kode-sumber)
6. [Tanya Jawab (FAQ) & Troubleshooting](#6-tanya-jawab-faq--troubleshooting)

---

## 1. Cara Menjalankan Mode UI Desktop

Aplikasi ini dilengkapi jendela desktop mandiri (*Windowed App Mode*). Tidak ada address bar peramban, tidak ada tab, dan tampilannya bersih layaknya software Windows biasa.

### Cara A: Dobel-Klik Langsung (Paling Mudah)
1. Buka folder tempat file `anyflip-downloader.exe` berada di **File Explorer**.
2. **Dobel-klik** file `anyflip-downloader.exe`.
3. Jendela aplikasi akan otomatis terbuka di layar komputer Anda.

### Cara B: Melalui Terminal (PowerShell / Command Prompt)
Buka terminal di folder aplikasi, lalu jalankan:
```powershell
.\anyflip-downloader.exe -ui
```
*(Menjalankan `.\anyflip-downloader.exe` tanpa argumen apapun juga akan otomatis membuka UI).*

---

## 2. Fitur-Fitur pada Antarmuka (UI)

Saat jendela aplikasi terbuka, Anda akan menemukan bagian-bagian berikut:

| Bagian | Fungsi & Cara Pakai |
| :--- | :--- |
| **URL Buku AnyFlip** | Kolom untuk menempelkan tautan buku AnyFlip yang ingin diunduh (contoh: `https://online.anyflip.com/vpslm/ckfb/`). |
| **Judul Dokumen PDF** | *(Opsional)* Tentukan nama file PDF yang diinginkan. Jika dikosongkan, nama file akan otomatis mengikuti judul asli di AnyFlip. |
| **Opsi Lanjutan (⚙️)** | Klik untuk menampilkan pengaturan tambahan: <br>• **Jumlah Unduhan Paralel (Threads)**: Geser slider (1–16) untuk mempercepat proses download.<br>• **Batch Konversi (Chunk)**: Mengatur jumlah gambar yang diproses sekaligus (default: 10).<br>• **Pertahankan Folder Gambar Asli**: Centang jika Anda ingin menyimpan semua file gambar individual (`.webp`/`.jpg`) dan tidak ingin dihapus setelah PDF selesai dibuat. |
| **Tombol Unduh & Konversi** | Klik tombol kuning **"Mulai Unduh & Konversi ke PDF"** untuk memulai proses. |
| **Monitor Progres Real-Time** | Menampilkan bilah progres persentase, penghitung halaman yang sedang diunduh/dikonversi, serta status tahapan. |
| **Konsol Log** | Terminal mini di dalam aplikasi yang memperlihatkan detail teknis proses unduhan secara langsung. |
| **Tombol Aksi Cepat** | Muncul setelah proses selesai:<br>• **Buka PDF**: Langsung membuka file dokumen PDF.<br>• **Buka Folder**: Membuka folder lokasi penyimpanan di File Explorer. |
| **Buku Tersimpan** | Daftar riwayat file PDF yang ada di folder kerja beserta informasi ukuran file dan tanggal pembuatan. |

---

## 3. Cara Menjalankan Mode CLI (Terminal)

Bagi Anda yang menyukai otomatisasi, skrip batch, atau lebih nyaman menggunakan terminal:

### Unduhan Dasar:
```powershell
.\anyflip-downloader.exe "https://online.anyflip.com/vpslm/ckfb/"
```

### Menggunakan Multi-Threading (Mempercepat Unduhan):
Gunakan flag `-threads` (misal 4 atau 8 thread paralel):
```powershell
.\anyflip-downloader.exe -threads 4 "https://online.anyflip.com/vpslm/ckfb/"
```

### Menyimpan Gambar Asli (Tanpa Dihapus):
Secara default gambar sementara akan dihapus setelah PDF terbentuk. Tambahkan flag `-keep-download-folder` untuk mempertahankannya:
```powershell
.\anyflip-downloader.exe -keep-download-folder -threads 4 "https://online.anyflip.com/vpslm/ckfb/"
```

### Mengubah Judul File PDF:
```powershell
.\anyflip-downloader.exe -title "Buku_Saya" "https://online.anyflip.com/vpslm/ckfb/"
```

### Menentukan Lokasi Folder Gambar Sementara:
```powershell
.\anyflip-downloader.exe -temp-download-folder "folder_sementara" "https://online.anyflip.com/vpslm/ckfb/"
```

---

## 4. Tabel Parameter & Flag CLI

| Flag | Tipe | Nilai Bawaan | Keterangan |
| :--- | :--- | :--- | :--- |
| `-ui` | boolean | `false` | Menjalankan aplikasi dalam mode antarmuka desktop (UI). |
| `-threads` | integer | `1` | Jumlah proses unduhan halaman paralel secara bersamaan. |
| `-keep-download-folder` | boolean | `false` | Mempertahankan folder gambar individual setelah konversi selesai. |
| `-title` | string | `""` | Menentukan nama dokumen PDF (otomatis mengambil judul AnyFlip jika kosong). |
| `-chunksize` | integer | `10` | Jumlah gambar yang dikonversi per batch (semakin tinggi = lebih cepat tapi makan RAM). |
| `-retries` | integer | `1` | Jumlah percobaan ulang jika pengunduhan suatu halaman gagal. |
| `-waitretry` | duration| `1s` | Jeda waktu sebelum mencoba ulang unduhan yang gagal (misal: `500ms`, `2s`). |
| `-temp-download-folder` | string | `""` | Menentukan nama folder penyimpanan gambar sementara secara spesifik. |
| `-insecure` | boolean | `false` | Menonaktifkan verifikasi sertifikat SSL/TLS jika diperlukan. |

---

## 5. Kompilasi / Build Ulang dari Kode Sumber

Jika Anda melakukan modifikasi pada kode Go ([`main.go`](file:///d:/git/anyflip-downloader/main.go), [`server.go`](file:///d:/git/anyflip-downloader/server.go), [`anyflip.go`](file:///d:/git/anyflip-downloader/anyflip.go)) atau tampilan antarmuka ([`web/index.html`](file:///d:/git/anyflip-downloader/web/index.html)):

1. **Jalankan langsung saat pengembangan:**
   ```powershell
   go run . -ui
   ```

2. **Kompilasi ulang menjadi file `.exe` mandiri:**
   ```powershell
   go build -o anyflip-downloader.exe .
   ```
   *File aset HTML/JS/CSS otomatis dikemas ke dalam biner biner `.exe` baru.*

---

## 6. Tanya Jawab (FAQ) & Troubleshooting

#### Q: Di mana hasil file PDF disimpan?
> **A:** File `.pdf` akan disimpan langsung di direktori yang sama dengan tempat Anda menjalankan aplikasi (folder kerja saat ini).

#### Q: Mengapa ada proses "Konversi ke PDF"?
> **A:** Server AnyFlip tidak menyediakan file PDF utuh secara publik; mereka menyimpannya per lembar halaman (format `.webp` atau `.jpg`). Aplikasi ini mengunduh setiap halaman tersebut lalu menyusunnya menjadi dokumen PDF utuh menggunakan engine `pdfcpu`.

#### Q: Apakah butuh koneksi internet untuk menjalankan antarmukanya?
> **A:** Tidak untuk antarmukanya. Server UI berjalan 100% lokal di komputer Anda (`127.0.0.1`). Koneksi internet hanya dibutuhkan saat mengunduh gambar buku dari AnyFlip.

#### Q: Bagaimana jika komputer saya tidak memiliki Microsoft Edge?
> **A:** Aplikasi memiliki deteksi cerdas: jika Microsoft Edge tidak ditemukan, aplikasi akan mencoba Google Chrome. Jika keduanya tidak tersedia, aplikasi otomatis memanggil peramban web bawaan sistem operasi Anda.
