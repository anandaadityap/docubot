# FAQ Support — Toko Kita (demo knowledge base)

Dokumen ini untuk diunggah ke DocuBot. Bot akan menjawab hanya dari isi di bawah.

---

## Jam operasional & kontak

- Chat bot aktif 24 jam.
- Tim manusia: Senin–Jumat, 09.00–17.00 WIB.
- Email support: support@tokokita.example
- WhatsApp (manusia): 0812-0000-0000 (jam kerja saja).
- Balasan email biasanya dalam 1 hari kerja.

## Cara daftar akun

1. Buka halaman Register.
2. Isi nama, email, dan password (minimal 8 karakter).
3. Cek kotak masuk untuk tautan verifikasi.
4. Klik tautan, lalu login.

Satu email hanya boleh satu akun. Kalau email sudah terpakai, login atau reset password.

## Cara login

1. Buka halaman Login.
2. Masukkan email dan password.
3. Jika lupa password, pakai alur reset password (bukan daftar ulang).

## Cara reset password

1. Buka **Settings** → **Security**.
2. Klik **Reset password**.
3. Masukkan email terdaftar.
4. Cek kotak masuk (termasuk folder spam) untuk tautan reset. Tautan berlaku 30 menit.
5. Buat password baru minimal 8 karakter, campur huruf dan angka.

Kalau email tidak masuk, pastikan alamat email benar lalu minta tautan baru. Jangan bagikan tautan reset ke orang lain.

## Paket & harga

| Paket | Harga / bulan | Kuota chat / bulan | Yang didapat |
|-------|---------------|--------------------|--------------|
| Starter | Rp 99.000 | 1.000 | 1 bot, upload dokumen 20 MB |
| Pro | Rp 249.000 | 10.000 | 3 bot, upload 100 MB, analytics |
| Business | Custom | Tidak terbatas | SLA, onboarding, branding |

Harga belum termasuk PPN 11%. Pembayaran otomatis tiap tanggal yang sama dengan tanggal langganan.

## Cara ganti paket

1. Login sebagai admin.
2. Buka **Setelan** → **Langganan**.
3. Pilih paket baru.
4. Upgrade berlaku langsung. Downgrade berlaku di siklus tagihan berikutnya.
5. Sisa kuota Starter tidak hangus saat naik ke Pro di bulan yang sama.

## Cara bayar

Metode yang diterima: transfer bank (BCA, Mandiri, BRI), QRIS, dan kartu Visa/Mastercard.

Setelah transfer manual, unggah bukti ke halaman **Tagihan**. Status berubah `lunas` paling lambat 1×24 jam hari kerja. Invoice dikirim ke email akun.

## Kebijakan refund

- Refund **penuh** dalam **7 hari** setelah pembelian pertama, jika produk belum dipakai (belum ada chat customer dan belum ada dokumen `ready`).
- Setelah 7 hari, refund **50%** hanya untuk bug kritis yang belum kami perbaiki dalam 14 hari.
- Langganan yang sudah dipakai lebih dari 7 hari tidak bisa di-refund (bisa berhenti di akhir periode).
- Ajukan refund ke support@tokokita.example dengan nomor invoice dan alasan singkat.
- Dana dikembalikan ke rekening/kartu asal dalam 5–10 hari kerja setelah disetujui.

## Cara upload dokumen knowledge base

1. Login sebagai admin.
2. Buka halaman **Dokumen**.
3. Unggah file `.md` atau `.txt` (maks 5 MB per file).
4. Tunggu status menjadi **ready**. Jangan hapus file sebelum status ready.
5. Kalau status **failed**, baca pesan error, perbaiki file, lalu **Proses ulang** atau unggah ulang.

Bot hanya menjawab dari dokumen berstatus ready. PDF dan file hasil scan belum didukung.

## Bot tidak menjawab / jawaban salah

- Pastikan ada dokumen status `ready`.
- Matikan lalu nyalakan bot di **Setelan** (centang **Bot aktif**).
- Pertanyaan di luar dokumen akan dijawab “tidak tahu” — itu disengaja, bot dilarang mengarang.
- Klik kutipan `[1]` `[2]` untuk cek sumber.
- Kalau jawaban meleset, perbarui dokumen lalu proses ulang.

## Bot sedang tidak aktif

Jika pengunjung melihat “Bot sedang tidak aktif”, admin telah mematikan bot di Setelan. Nyalakan kembali centang **Bot aktif**, lalu simpan.

## Privasi data

- Percakapan customer disimpan di dashboard admin (halaman Percakapan).
- Dokumen knowledge base hanya dipakai untuk menjawab chat, tidak dijual ke pihak ketiga.
- Hapus dokumen = hapus juga potongan teks (chunk) terkait.
- Untuk hapus akun, email support@tokokita.example dari email terdaftar. Penghapusan selesai dalam 7 hari kerja.

## Pengiriman (jika beli add-on fisik)

- Jabodetabek: 1–2 hari kerja.
- Kota besar lain: 2–4 hari kerja.
- Luar Jawa: 4–7 hari kerja.
- Resi dikirim ke email setelah paket diserahkan ke kurir.
- Kerusakan saat kirim: foto unboxing dalam 24 jam, klaim ke support dengan nomor resi.

## Pertanyaan yang tidak dijawab bot

Hal-hal ini harus ke manusia (email/WhatsApp jam kerja):

- sengketa pembayaran yang sudah lebih dari 14 hari
- permintaan data pribadi orang lain
- kerja sama / partnership
- lowongan kerja
