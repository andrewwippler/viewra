# Live TV Setup Guide

StreamJuice supports Live TV via IPTV playlists (M3U) with optional Electronic Program Guide (EPG) data from XMLTV files. It handles SAT>IP tuners, RTSP streams, and standard HTTP/HLS streaming sources, transcoding everything on-the-fly to HLS for browser playback with DVR support.

## Prerequisites

- **FFmpeg** installed on the server (required for transcoding)
- **An M3U playlist** from an IPTV provider or SAT>IP tuner
- **(Optional) An XMLTV EPG file** for program guide data

## Quick Start

### 1. Get your M3U playlist

An M3U file contains channel definitions with stream URLs. It typically looks like:

```
#EXTM3U
#EXTINF:-1 tvg-id="cnn" tvg-logo="https://example.com/cnn.png" group-title="News",CNN HD
http://example.com/stream/cnn.ts
#EXTINF:-1 tvg-id="hbo" tvg-logo="https://example.com/hbo.png" group-title="Movies" tvg-chno="5",HBO
http://example.com/stream/hbo.ts
```

Your M3U can be:
- A local file (e.g., `/home/user/iptv/playlist.m3u`)
- A local directory containing one or more `.m3u` files
- A remote URL (e.g., `http://iptv-provider.com/get.php?username=...&type=m3u`)

### 2. (Optional) Get an XMLTV EPG file

An XMLTV file contains program schedule data. It typically looks like:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<tv generator-info-name="generator">
  <channel id="cnn">
    <display-name>CNN HD</display-name>
  </channel>
  <programme start="20260101000000 +0000" stop="20260101010000 +0000" channel="cnn">
    <title>CNN Newsroom</title>
    <desc>Live news coverage.</desc>
    <category>News</category>
  </programme>
</tv>
```

Your EPG can be:
- A local file named `guide.xml` placed in the same directory as your M3U files (auto-detected during scan)
- Any local `.xml` or `.xmltv` file
- A remote URL (e.g., `http://epg-provider.com/guide.xml`)

### 3. Create a Live TV library

1. Navigate to **Libraries** in the web UI
2. Click **Add Library**
3. Select type **Live TV**
4. Give it a name (e.g., "My IPTV")
5. Set the path to your M3U file, directory, or paste the M3U URL
6. (Optional) Set an XMLTV URL in the library's settings for EPG data
7. Save

### 4. Scan channels

In the Live TV page, click **Scan Channels**. This reads your M3U file(s), parses all channels, and stores them.

**If you use SAT>IP:** The scan automatically detects SAT>IP channellist URLs (containing `/dvb/m3u/`) and corrects PID parameters against the authoritative tuner list. PID corrections are also refreshed monthly via a scheduled task.

### 5. Scan EPG (optional but recommended)

Click **Scan EPG** to import program guide data. This reads the XMLTV source (from the configured URL, an auto-detected `guide.xml` in the library folder, or a manually set URL in library settings) and maps programs to channels.

### 6. Map channels (if needed)

If your M3U `tvg-id` attributes don't match the XMLTV channel IDs, go to the **Mappings** page. You'll see each XMLTV guide channel with a dropdown to link it to the correct M3U channel. This step is only needed when automatic matching fails.

### 7. Watch live TV

Go to the Live TV page to see the EPG grid:

- **Channel sidebar** (left): channel numbers, names, and current programs
- **Program grid**: time-based blocks showing what's on, color-coded by category (news, sports, movies, etc.)
- **Red "now" line**: indicates the current time
- Click a channel to see details; double-click to start watching

Click **Watch Now** on any channel to open the full-screen player:

- **Play/Pause**: Pausing enables DVR — you can seek back up to 60 minutes
- **Stop**: Stops the stream and cleans up the FFmpeg process
- **Seek bar**: Navigate through available DVR buffer
- **Volume/fullscreen**: Standard player controls

## DVR (Digital Video Recorder)

StreamJuice provides a 60-minute DVR buffer for every channel:

- **Pause**: Freezes playback. The FFmpeg process keeps running, buffering up to 60 minutes of video. Only one channel can be paused at a time globally.
- **Resume**: Continues playing from the paused position. You can seek back and forth within the DVR window.
- **Stop**: Terminates the transcode process and frees resources.

## Scheduled Tasks

| Task | Schedule | Description |
|------|----------|-------------|
| EPG refresh | Daily at 4:00 AM | Refreshes EPG data for all libraries with an XMLTV URL configured |
| SAT>IP PID sync | Monthly on 1st at 3:00 AM | Refreshes PID values from SAT>IP tuner for channels using SAT>IP sources |

## Technical Details

### Transcoding

Every channel is transcoded to HLS with a single quality profile:

| Setting | Value |
|---------|-------|
| Resolution | 1920x1080 |
| Video codec | H.264 (libx264, `veryfast` preset) |
| Video bitrate | 5 Mbps, CRF 23 |
| Audio codec | AAC, 128 kbps, 48 kHz, stereo |
| Segment duration | 2 seconds |
| DVR window | 1800 segments (~60 minutes) |
| Playlist type | `event` (growing playlist) |

RTSP streams use a two-process FFmpeg pipeline: a producer that fetches the stream and writes to a FIFO, and a transcoder that reads from the FIFO and outputs HLS. HTTP/HLS streams use a single FFmpeg process with the `-re` flag for real-time playback.

### Data storage

- Channel and EPG data is stored in the database
- HLS segments are written to `{data_dir}/livetv_{libraryId}_{channelId}/` and cleaned up when the stream stops
- At 5 Mbps, a single channel consumes approximately 1.25 GB/hour of disk space for the DVR buffer

## Troubleshooting

| Symptom | Likely cause |
|---------|-------------|
| No channels after scan | M3U path is incorrect or file is unreadable |
| EPG scan shows 0 programs | XMLTV URL is not set or `guide.xml` is not in the library directory |
| Stream won't play | FFmpeg is not installed or the stream URL is unreachable |
| "Another stream is paused" | Only one channel can be paused at a time globally — stop the other paused stream first |
| No EPG data for a channel | M3U `tvg-id` doesn't match XMLTV channel ID — use the Mappings page to link them manually |
