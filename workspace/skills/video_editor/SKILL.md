# Video Editor Skill

This skill automates the process of retrieving video footage, generating subtitles, and performing logical edits based on audio and caption content.

## Workflow

1. **Retrieve Footage**:
   - Use `vps_exec` to run `gws download <file_id>` on the VPS.
   - Verify the file is downloaded to the VPS workspace.

2. **Isolate Audio**:
   - Use `vps_exec` with `ffmpeg` to extract audio:
     `ffmpeg -i input.mp4 -vn -acodec pcm_s16le -ar 16000 -ac 1 output.wav`

3. **Generate Subtitles (.srt)**:
   - **For Long Gameplay (2-5 hours)**: Do NOT use managed APIs due to duration limits and costs. Use local offline processing.
   - Run `whisper.cpp` (compiled locally) or the standalone Python `whisper` CLI on the VPS:

     ```bash
     # Example using whisper CLI. Consider chunking the WAV if memory is constrained.
     vps_exec "whisper output.wav --model base --output_format srt --language en"
     ```

   - *Optimization Note*: If the VPS struggles with memory on a 5-hour file, use `ffmpeg` to split the audio into 30-minute chunks, transcribe them, and concatenate the `.srt` files adjusting timestamps accordingly.
   - Clean up subtitles if needed for Vegas 23 compatibility.

4. **Logical Cutting**:
   - Analyze the `.srt` and audio for silence or specific keywords (e.g., "boss", "death").
   - Generate complex `ffmpeg` filter scripts or concat files based on timestamps.
   - Taxonomy for markers:
     - Boss Fights: `start_time`, `end_time`, `boss_name`, `result`.
     - Player Death: `timestamp`, `cause_of_death`.
     - Progression: `timestamp`, `item_acquired / milestone_reached`.
     - Transitions: `timestamp`, `biome_change`.
     - Events: `timestamp`, `event_name`.

5. **Execute Edits**:
   - Run the final `ffmpeg` command on the VPS to produce the edited video and audio package.

6. **Upload to Google Drive**:
   - Use `vps_exec` to run `gws upload <edited_video>` to sync the results back to Google Drive.

## Tools Used

- `vps_exec`: For running `gws`, `ffmpeg`, and `whisper` on the high-compute VPS.
- `google`: For listing Drive files if needed (though `gws` is preferred on VPS).
- `read_file` / `write_file`: For local log/metadata management.
