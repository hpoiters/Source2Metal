SyzygyCheck v2.0.9 V9F — All languages
=====================================

Purpose
-------
SyzygyCheck verifies Syzygy .rtbw/.rtbz files located in the same folder as the EXE. Parent folders and
subfolders are not scanned.

Verification core and credit
----------------------------
Based on the original Syzygy tablebase verification code by Ronald de Man.
The AL layer does not reimplement the checksum logic. The multilingual interface, progress display, self-test,
checkpoint/resume, thread selection and reporting are built around that verification core.
Upstream: https://github.com/syzygy1/tb

Thread selection
----------------
At each start SyzygyCheck silently detects the logical processors Windows makes available to the process.
It then offers:

  1 = use all detected threads
  2 = use half [default; Enter selects this]
  3 = choose a custom value from 1 to the detected maximum

The selected value is shown as `Workers (threads): <count>` and is passed directly to the original tbcheck as
`--threads n`. Half is the balanced default; all threads or a custom count can be used for throughput testing.
Esc goes back from the thread menu; in custom-count entry Esc returns one level to the thread menu.

Pagefile / virtual memory
-------------------------
Very large checks can require substantial virtual memory and run for many hours. Before the self-test, the
program therefore shows a short warning. Enter continues; A/Esc goes back so Windows settings can be reviewed first.
Detailed PAGEFILE_GUIDE_XX.txt files are included in all seven supported languages. Windows-managed pagefile
sizing is the recommended simple starting point for most systems. SyzygyCheck never changes the setting itself.

Onboard diagnostic self-test
----------------------------
A stale reserved self-test file is silently removed at startup. A fresh tiny onboard file with an intentional
checksum mismatch is then written temporarily, checked by the tbcheck by Ronald de Man, and removed before any real
.rtbw/.rtbz inventory begins. The user's own tablebases are not used or modified for this diagnostic. A failed
self-test prevents the real scan; a successful one pauses for Enter before the real scan starts, while Esc returns
to the main menu.

Progress and error visibility
-----------------------------
The live status is a compact resize-safe three-line block:

  Checking file: n/total        Read speed: xx.x MB/s
  [adaptive byte-volume bar]
  percentage   processed/total  Remaining: ...

Where console width permits, `Read speed` and `Remaining` start in the same right-hand column. After a visible
FAIL/ERROR there is exactly one blank separator before the live block. With many errors, current progress remains
visible at the bottom while older error lines may scroll upward and remain available in console scrollback.

The displayed read speed is effective verification throughput for the current session, not necessarily raw
physical disk or bus speed. While a tbcheck batch is running, the live display uses the Windows process-I/O
counter as a progress estimate. After file results are confirmed, throughput is based on definitively verified
bytes. The final report uses definitively verified bytes for the current session divided by elapsed check time.
Bytes completed in an earlier session are excluded after resume.

Data volume is the primary progress measure; file number is supplementary. The bar uses as much of the current
console width as possible while leaving one console cell unused to avoid automatic wrapping.

FAIL and ERROR
--------------
FAIL means an embedded checksum is present but does not match the recalculated checksum. ERROR means a file could
not be checked reliably because of a read, structure or process problem. One problematic file must not hide the
result of later files.

Window close protection during the real scan
--------------------------------------------
During a real Syzygy scan, SyzygyCheck makes a best-effort attempt to disable SC_CLOSE for a classic Windows
console host. In Windows Terminal, however, the outer top-right X belongs to Windows Terminal itself and can still
close the window. Minimize, maximize, resizing and scrollback are not intentionally blocked. Checkpoint/resume
remains the dependable protection against losing already confirmed progress after an accidental close.

Checkpoint / resume
-------------------
A small checkpoint is updated after confirmed file outcomes. An interrupted large check can resume if filename,
size and modification time still match. Confirmed FAIL results already stored in the checkpoint remain available
and are carried into the resumed run. A batch that was not definitively completed may be checked again.

A fully completed run without operational ERROR removes the checkpoint. During the real scan SyzygyCheck asks
Windows to prevent system sleep temporarily; this request is released after the run.

Reports
-------
The result report is normally written to the Windows Desktop, with the full path also shown. Normal OK files are
not listed individually; only summary information, real checksum FAILs and operational ERRORs are listed.
The average effective read speed of the current session is shown on the same line as run duration.
Report names use date and hour/minute:
SyzygyCheck_Result_YYYYMMDD_HHMM.txt, with _2, _3, ... only on a rare collision.

Recovery after FAIL/ERROR
-------------------------
When a real checksum FAIL or operational ERROR is found, the desktop report refers to the supplied
SYZYGY_RECOVERY_EN.txt. Equivalent recovery guides are included in all seven supported languages. They contain a
short replacement/recheck procedure and two download sources: the Lichess Syzygy mirror and the Sesse mirror.

Languages
---------
English, German, Dutch, French, Spanish, Chinese and Russian are supported. The selected language also applies to
the report. Main-menu language switching always includes recognisable English/Chinese/Russian bridge labels.
The physical Esc key is also a genuine back action in choice screens: on the main menu it opens language choice;
in language, pagefile, thread and resume choices it returns without starting a scan.

Ronald helper
-------------
Reference tbcheck.exe SHA-256:
9cb2f0ba2343f343bab29f9b9cd6aad6e4a2612702a93e06aa84a417f8eabbcc
If that exact helper is already next to SyzygyCheck it is used. Otherwise v2.0.9 temporarily retrieves the fixed
v3.0.7 copy, verifies its SHA-256 and removes it after the run. From the user's perspective SyzygyCheck remains a
single EXE.

Practical validation / status
-----------------------------
V9F builds on the successfully tested V9C/V9D/V9E line. Verification and thread behaviour are unchanged.
The main-menu prompt is quieter in all seven languages, and hour units are localised as `ч (h)` in Russian and
`小时 (h)` in Chinese. The V9E SC_CLOSE measure remains a best-effort classic-console safeguard only; the outer
X of Windows Terminal belongs to Terminal itself and can still close the window. Checkpoint/resume remains the
dependable protection against losing confirmed progress.

V9C practical results carried forward into V9F:
- i7-6800K: 12 workers, 57 files / 20.22 GB, 27 s after reboot and 805.0 MB/s average; the two intentional
  checksum failures were detected and there were 0 read/process errors.
- Drobo: verification also works well there; 4 threads produced reasonably normal mechanical behaviour and
  practically acceptable throughput.
- The large X299 test on H:\W1 with 1511 RTBW files on 5× NVMe RAID0 remains useful as a scaling test, but the
  already completed functional validation does not block preserving/publishing this development state.
