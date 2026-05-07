# Penyahpepijatan PicoClaw

PicoClaw melakukan pelbagai interaksi kompleks di sebalik tabir untuk setiap permintaan yang diterimanya, daripada menghala mesej dan menilai kerumitan, hinggalah melaksanakan tools dan menyesuaikan diri dengan kegagalan model. Keupayaan melihat dengan tepat apa yang sedang berlaku sangat penting, bukan sahaja untuk menyelesaikan masalah, malah untuk benar-benar memahami cara agen ini beroperasi.
## Memulakan PicoClaw dalam Mod Debug

Untuk mendapatkan maklumat terperinci tentang apa yang sedang dilakukan oleh agen (permintaan LLM, panggilan tool, penghalaan mesej), anda boleh memulakan gateway PicoClaw dengan flag debug:

```bash
picoclaw gateway --debug
# or
picoclaw gateway -d
```

Dalam mod ini, sistem akan memformat log dengan lebih terperinci dan memaparkan pratonton system prompt serta hasil pelaksanaan tool.

## Menyahaktifkan Pemotongan Log (Log Penuh)

Secara lalai, PicoClaw memotong rentetan yang sangat panjang (seperti *System Prompt* atau hasil output JSON yang besar) dalam log debug supaya konsol kekal mudah dibaca.

Jika anda perlu memeriksa output penuh sesuatu arahan atau payload tepat yang dihantar kepada model LLM, anda boleh menggunakan flag `--no-truncate`.

**Nota:** Flag ini *hanya* berfungsi apabila digabungkan dengan mod `--debug`.

```bash
picoclaw gateway --debug --no-truncate

```

Apabila flag ini aktif, fungsi pemotongan global dinyahaktifkan. Ini sangat berguna untuk:

* Mengesahkan sintaks tepat mesej yang dihantar kepada penyedia.
* Membaca output lengkap daripada tools seperti `exec`, `web_fetch`, atau `read_file`.
* Menyahpepijat sejarah sesi yang disimpan dalam memori.
