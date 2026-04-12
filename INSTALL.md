# NIL — Installation Guide

Welcome to the NIL beta! Below are step-by-step instructions for each platform.

---

## macOS

### Installing from the DMG

1. Open the `.dmg` file you received.
2. Drag **NIL.app** into the **Applications** shortcut in the window.
3. Eject the disk image.

### First launch (Gatekeeper bypass)

Because this build is not signed with an Apple Developer certificate, macOS will
block a normal double-click the first time.

**Right-click → Open** instead:

1. Open **Finder** and go to **Applications**.
2. Right-click (or Control-click) **NIL.app**.
3. Choose **Open** from the context menu.
4. Click **Open** in the dialog that appears.

After you do this once, future launches work normally.

### "NIL is damaged and can't be opened"

If you see this message, the quarantine attribute needs to be cleared. Open
**Terminal** and run:

```bash
xattr -cr /Applications/NIL.app
```

Then try launching again via right-click → Open.

### Your data

All data is stored locally on your Mac:

```
~/Library/Application Support/Nil/
```

Nothing is sent to any server.

---

## Windows

### Running the installer

1. Double-click **NIL-1.2.1-Windows-Setup.exe**.
2. If Windows SmartScreen appears with "Windows protected your PC":
   - Click **More info**.
   - Click **Run anyway**.
3. Follow the setup wizard (Next → Install → Finish).

The installer creates:
- A desktop shortcut
- A Start Menu entry under **NIL**
- An entry in **Add or Remove Programs** for clean uninstalls

### Your data

All data is stored locally on your PC:

```
%APPDATA%\Nil\
```

(Paste that path into File Explorer's address bar to open it.)

Nothing is sent to any server.

---

## Linux

### AppImage (recommended)

If you received a `.AppImage` file:

1. Make it executable and run it:

```bash
chmod +x NIL-1.2.1-Linux.AppImage
./NIL-1.2.1-Linux.AppImage
```

No installation required — it runs in place. You can move it anywhere (e.g. `~/Applications/`).

### tar.gz (fallback)

If you received a `.tar.gz` file:

```bash
tar xzf NIL-1.2.1-Linux.tar.gz
./nil
```

To add it to your app launcher, create `~/.local/share/applications/nil.desktop`:

```ini
[Desktop Entry]
Name=NIL
Exec=/path/to/nil
Icon=/path/to/icon.png
Type=Application
Categories=Productivity;
```

### Your data

All data is stored locally:

```
~/.local/share/nil/
```

Nothing is sent to any server.

---

## Feedback

Found a bug or have a suggestion? Open an issue on GitHub — this is a beta and your
feedback is very welcome.
