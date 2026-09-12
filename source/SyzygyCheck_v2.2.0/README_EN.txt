SyzygyCheck v2.2.0 — All languages
=====================================

Purpose
-------
This documentation calls the folder containing !SyzygyCheck_v2.2.0.exe the control folder. For every run the user chooses:

  1 = this control folder only [default]
  2 = this control folder and all subfolders

Parent folders are never scanned. Option 2 includes real subfolders to any depth. Symbolic links and Windows
folder links are not followed when they could lead outside the control folder.
When the control folder is a drive root, Windows volume metadata folders `$RECYCLE.BIN` and
`System Volume Information` are skipped. Deleted or protected files there are not part of the active tablebase set.

Release package
---------------
The complete user release is one ZIP with four fixed items in its root:

  !SyzygyCheck_v2.2.0.exe
  1_READ_FIRST_AL.txt
  Docs_AL\
  Program_AL\

Keep these items together. `1_READ_FIRST_AL.txt` provides the short start instructions in all seven languages and
points to `Docs_AL\README_AL.txt`. Program_AL contains the seven recovery guides named by exact path in a
FAIL/ERROR report. Source is not included in the user package. The recommended source download is the clearly named
`SyzygyCheck_v2.2.0_SOURCE.zip` release asset; the repository remains available at
https://github.com/hpoiters/SyzygyCheck.

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

After completion, exactly one blank line also separates the final percentage/remaining-time block from the prompt
asking where to save the result report. This keeps two different interface sections visually distinct.

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
Immediately before saving, SyzygyCheck asks where the result report should be written:
1. the Windows Desktop [default on first use];
2. the control folder;
3. a freely selected folder through the Windows Explorer-style folder dialog.

Inside the selected location SyzygyCheck automatically creates the collection folder CheckResult. Every later
report is stored there, keeping checks of a large database spread over several folders from scattering separate
result files across the Desktop. Chinese and Russian use a translated folder name in their own script. The last
selected destination is remembered and becomes the default for the next run; Enter then uses it immediately. An
exact path is remembered for a freely selected folder. If that folder no longer exists or is unavailable,
SyzygyCheck displays a warning with the unavailable previous path and explains that Desktop is now the default.
The warning is a separate visual paragraph before the destination menu. Pressing Enter then saves to the Desktop
collection folder and replaces the invalid remembered preference. The final screen always shows the full saved path. Normal OK files are
not listed individually; only summary information, real checksum FAILs and operational ERRORs are listed.
These choice screens are resize-safe too: lines and long remembered paths wrap to the current console width,
with CJK characters counted by their actual console-cell width.
Every FAIL and ERROR includes the full Windows path from the drive letter, making the exact location immediately
recognisable when one run covers several subfolders.
The average effective read speed of the current session is shown on the same line as run duration.
Report names use date and hour/minute:
SyzygyCheck_Result_YYYYMMDD_HHMM.txt, with _2, _3, ... only on a rare collision.

Recovery after FAIL/ERROR
-------------------------
When a real checksum FAIL or operational ERROR is found, the result report refers to the supplied
`Program_AL\SYZYGY_RECOVERY_EN.txt`. Equivalent recovery guides are included in Program_AL for all seven supported
languages. They contain a
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
If that exact helper is already next to SyzygyCheck it is used. Otherwise v2.2.0 temporarily retrieves the fixed
v3.0.7 copy, verifies its SHA-256 and removes it after the run. From the user's perspective SyzygyCheck remains a
single verification EXE; the user release additionally contains the start text, documentation and recovery guides
required to make every screen and report reference usable.

Practical validation / status
-----------------------------
v2.2.0 builds on v2.1.0 and the successfully tested v2.0.9 V9F. Verification and thread behaviour are unchanged.
The new layer adds explicit subfolder scope, a hard control-folder boundary and full problem paths. V9F remains
available untouched as the first practically validated release.
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
