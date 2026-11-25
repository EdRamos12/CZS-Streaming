# CZS Streaming

Live stream overlay and control panel for **Clube Zona Sul** Riichi Mahjong tournaments, built for OBS and Mahjong Soul.

A tiny Go server (standard library only, no dependencies) serves a control panel and a set of browser-source overlays. Everything you change in the panel shows up live on stream through Server-Sent Events.

## Features
- In-game overlay: player names, scores, seat winds, round, honba and riichi sticks, placements
- Win / draw dialog that does the scoring for you: ron, tsumo, double/triple ron, exhaustive draw, with a preview before applying
- Riichi call animation, riichi stick markers and wait (tenpai) tiles per player
- Undo for the last hand
- Full-screen ranking / break screen with standings that scroll on their own
- Commentary scene with camera frames and a chat slot
- Sponsor and banner rotation with image uploads
- State saved to disk, so a restart keeps everything

## Quick start
Needs Go 1.22 or newer (Node only for the JS tests).

    go run .              # then open http://localhost:8787/
    go build              # or build a single binary: ./czs-livestreams (run it from this folder)
    go test               # server self-check
    node test.js          # scoring/overlay logic self-check

| URL | What it is |
| --- | --- |
| `http://localhost:8787/` | Control panel |
| `/overlay.html` | In-game overlay (1920x1080) |
| `/ranking.html` | Break / ranking screen |
| `/cams.html` | Commentary scene |

## OBS setup (canvas 1920x1080)
1. **Window/Game Capture** of Mahjong Soul. Place it at 0,0 and size it to 1600x900.
2. **Browser source** above it: `http://localhost:8787/overlay.html`, 1920x1080.
3. Commentator cameras (optional): Video Capture sources under the browser source, inside the gold boxes. Pick 0, 1 or 2 cameras in the control panel; the commentators line sits under the first camera (with no camera it moves up under the title).
   - 1 camera: 1600,120 sized 320x180
   - 2 cameras: 1600,120 and 1600,336, both 320x180
   The ranking screen ("Scene → Ranking" or `ranking.html`) puts its camera boxes in the same column as the commentary chat, so the camera sources must move there (see step 4).
4. Break/intermission scene: browser source `http://localhost:8787/ranking.html`, with the camera sources under it: 1 camera at 1403,123 sized 454x256; 2 cameras at 1403,123 and 1403,418, both 454x256.
5. Commentary scene (cameras + chat): browser source `http://localhost:8787/cams.html` in its own OBS scene.
   - Cameras go **under** the browser source: 1 camera at 80,220 sized 1280x720; 2 cameras at 60,330 and 720,330, both 640x360.
   - Chat goes **above** it (e.g. the YouTube/Twitch pop-out chat as a browser source with a transparent background): 1403,111 sized 454x906.
6. (Optional) Docks → Custom Browser Docks → `http://localhost:8787/` to run the control panel inside OBS.

The control panel has two pages (links at the top): **Game** (`/`) and **Banners & sponsors** (`/media.html`).

## Running a game
- List the four players in **starting seat order** (whoever is East at 東1 goes first). The seat winds then rotate on their own as you change the hand.
- After a win, press **Win**: pick the winner, Tsumo or Ron (and who dealt in), and the hand's value (the ron value; tsumo is split the standard way). Presets fill in mangan…yakuman for dealer or non-dealer. The preview shows every change; Apply pays honba and riichi sticks too, then moves to the next hand (or repeats the dealer).
- Exhaustive draw: press **Win / draw**, pick **Draw** and tick who was tenpai (riichi players are pre-ticked). Noten players pay 3000 split among tenpai players, honba goes up by 1, the dealer repeats if tenpai, and riichi sticks stay on the table for the next winner.
- Double/triple ron: pick **Ron**, tick every winner and type each one's hand value in the box under their name (presets fill the last winner you clicked). Honba and riichi sticks go to the winner closest after the one who dealt in; the dealer repeats if they were one of the winners.
- Abortive draws and other cases: copy the scores from Mahjong Soul by hand. Use "Next hand" or "Dealer repeats" for the round counter.
- "Starting points" sets what "Reset round" gives everyone (default 25,000).
- "Reset hand" undoes scores/round back to the start of the current hand. Pressed again right after, it goes back one more hand (to undo a misclicked Win). "Reset round" starts a new game (東1, 25,000 each).
- After each hanchan, update the standings box (`Name points`, one per line). If the list is longer than the box, it scrolls slowly on its own.
- "Riichi" next to a player takes their 1000, adds a stick to the table, plays the call on the overlay and shows a stick under their seat. Click again to undo a misclick. The stick markers clear on the next hand.
- "Waits" button next to a player: click the tiles they are waiting on (tenpai), up to 3. They show as tiles under the score on the overlay right away and clear on the next hand. Tile art: [FluffyStuff/riichi-mahjong-tiles](https://github.com/FluffyStuff/riichi-mahjong-tiles) (public domain).
- Scene section: "Round wind" shows the current round in Japanese (東1局), Portuguese (Leste 1) or romaji (Ton 1); seat winds stay in kanji. "Riichi call" picks the call animation text: リーチ or RIICHI.
- "Scene → Ranking" fades the full-screen ranking over the game, no OBS scene switch needed.
- **Banners & sponsors** page: upload images (png, jpg, webp, gif, up to 10 MB), reorder or remove them. Sponsors rotate with the club logo in the bottom-right corner; banners rotate on the break screen (the mascot shows when there are none). Files go to `public/uploads/`.

State is saved to `state.json`, so restarting the server keeps everything.
To run the control panel from another computer or phone on the LAN: `HOST=0.0.0.0 go run .`. This has no password, so only do it on a trusted network.

## Credits
Tile art: [FluffyStuff/riichi-mahjong-tiles](https://github.com/FluffyStuff/riichi-mahjong-tiles), public domain (CC0). Club logo and mascot art belong to Clube Zona Sul.
