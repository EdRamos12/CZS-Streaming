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
Needs Go 1.24 or newer (Node only for the JS tests).

    go run .              # then open http://localhost:8787/
    go build              # or build a single binary: ./czs-livestreams (run it from this folder)
    go test               # server self-check
    node test.js          # scoring/overlay logic self-check

**Password:** the first time you open the control panel it asks you to create a password (this has to be done on the computer running the server, unless you start it with `go run . -remote-setup`, which lets any computer on the network create it). After that, every browser has to type it once to use the panel. The OBS overlays don't need it. Forgot the password? Stop the server, run `go run . -reset-password` (or `./czs-livestreams -reset-password`, or just delete `password.json`), start it again and set a new one. Resetting logs out every browser. After 5 wrong passwords, that device has to wait 5 minutes before trying again.

| URL | What it is |
| --- | --- |
| `http://localhost:8787/` | Control panel |
| `/overlay.html` | In-game overlay (1920x1080) |
| `/ranking.html` | Break / ranking screen |
| `/cams1.html` | Commentary scene, 1 camera |
| `/cams2.html` | Same, 2 cameras |
| `/chat1.html` | In-game overlay with chat instead of the standings, 1 camera |
| `/chat2.html` | Same, 2 cameras |
| `/info.html` | Info scene: banners + your Markdown text + chat, no camera |

## OBS setup (canvas 1920x1080)
1. **Window/Game Capture** of Mahjong Soul. Place it at 0,0 and size it to 1600x900.
2. **Browser source** above it: `http://localhost:8787/overlay.html`, 1920x1080.
3. Commentator cameras (optional): Video Capture sources under the browser source, inside the gold boxes. Pick 0, 1 or 2 cameras in the control panel; the commentators line sits under the first camera (with no camera it moves up under the title).
   - 1 camera: 1600,120 sized 320x180
   - 2 cameras: 1600,120 and 1600,336, both 320x180
   The ranking screen ("Scene → Ranking" or `ranking.html`) puts its camera boxes in the same column as the commentary chat, so the camera sources must move there (see step 4).
4. Break/intermission scene: browser source `http://localhost:8787/ranking.html`, with the camera sources under it: 1 camera at 1403,123 sized 454x256; 2 cameras at 1403,123 and 1403,418, both 454x256.
5. Commentary scene (cameras + chat): browser source `http://localhost:8787/cams1.html` (1 camera) or `cams2.html` (2 cameras) in its own OBS scene. These pages ignore the panel's camera setting.
   - Cameras go **under** the browser source: 1 camera at 80,220 sized 1280x720; 2 cameras at 60,330 and 720,330, both 640x360.
   - Chat goes **above** it (e.g. the YouTube/Twitch pop-out chat as a browser source with a transparent background): 1403,111 sized 454x906.
6. In-game with chat (instead of the standings): use `chat1.html` (always 1 camera) or `chat2.html` (always 2 cameras) in place of `overlay.html`. Cameras go at the same spots as step 3. Put the chat browser source **above** it, under the "Chat" heading: about 1600,380 sized 320x516 (1 camera) or 1600,596 sized 320x300 (2 cameras). These pages ignore the panel's camera setting.
7. Info scene: browser source `http://localhost:8787/info.html` in its own OBS scene, with the chat source above it at the same spot as step 5 (1403,111 sized 454x906). No camera. Its images and text are set on the **Banners & sponsors** page.
8. (Optional) Docks → Custom Browser Docks → `http://localhost:8787/` to run the control panel inside OBS.

The control panel has two pages (links at the top): **Game** (`/`) and **Banners & sponsors** (`/media.html`). On the Game page the live controls (round, table, Win / draw) are on the left and the broadcast setup on the right; on a narrow window (e.g. an OBS dock) they stack, live controls first. The dealer's seat is highlighted.

## Running a game
- List the four players in **starting seat order** (whoever is East at 東1 goes first). The seat winds then rotate on their own as you change the hand.
- After a win, press **Win**: pick the winner, Tsumo or Ron (and who dealt in), and the hand's value (the ron value; tsumo is split the standard way). Presets fill in mangan…yakuman for dealer or non-dealer. The preview shows every change; Apply pays honba and riichi sticks too, then moves to the next hand (or repeats the dealer).
- Exhaustive draw: press **Win / draw**, pick **Draw** and tick who was tenpai (riichi players are pre-ticked). Noten players pay 3000 split among tenpai players, honba goes up by 1, the dealer repeats if tenpai, and riichi sticks stay on the table for the next winner.
- Double/triple ron: pick **Ron**, tick every winner and type each one's hand value in the box under their name (presets fill the last winner you clicked). Honba and riichi sticks go to the winner closest after the one who dealt in; the dealer repeats if they were one of the winners.
- Abortive draws and other cases: copy the scores from Mahjong Soul by hand. Use "Next hand" or "Dealer repeats" for the round counter.
- "Starting points" (Scene section) sets what "New game" gives everyone (default 25,000).
- "Undo hand" undoes scores/round back to the start of the current hand. Pressed again right after, it goes back one more hand (to undo a misclicked Win). "New game" starts a new game (東1, 25,000 each). Hover any round button for a reminder of what it does.
- After each hanchan, update the standings box (`Name points`, one per line). If the list is longer than the box, it scrolls slowly on its own.
- "Riichi" next to a player takes their 1000, adds a stick to the table, plays the call on the overlay and shows a stick under their seat. Click again to undo a misclick. The stick markers clear on the next hand.
- "Waits" button next to a player: click the tiles they are waiting on (tenpai), up to 3. They show as tiles under the score on the overlay right away and clear on the next hand. Tile art: [FluffyStuff/riichi-mahjong-tiles](https://github.com/FluffyStuff/riichi-mahjong-tiles) (public domain).
- Scene section: "Round wind" shows the current round in Japanese (東1局), Portuguese (Leste 1) or romaji (Ton 1); seat winds stay in kanji. "Riichi call" picks the call animation text: リーチ or RIICHI.
- "Scene → Ranking" fades the full-screen ranking over the game, no OBS scene switch needed.
- **Banners & sponsors** page: upload images (png, jpg, webp, gif, up to 10 MB), reorder or remove them. Sponsors rotate with the club logo in the bottom-right corner; banners rotate on the break screen (the mascot shows when there are none). **Info screen banners** and **Info screen text** (same page) feed `info.html`: the images rotate on the left, the text shows on the right as you type. The text takes simple Markdown (`# Heading`, `**bold**`, `*italic*`, `` `code` ``, `- item`, `1. item`, `- [ ] todo` / `- [x] done`, blank line between paragraphs) and scrolls on its own if it is too long; left empty, the banner takes the whole box. With no info images it falls back to the break banners, then the mascot. Files go to `public/uploads/`.

State is saved to `state.json`, so restarting the server keeps everything.
The control panel is reachable from other computers and phones on the LAN by default (it is password protected). To allow only this computer: `HOST=127.0.0.1 go run .`.

## License & credits
The code is licensed under the [GNU General Public License v3.0](LICENSE).

The club logo and mascot artwork (`public/assets/logo.png`, `icon.png`, `pexe.png`, `nanda.png`, `ponte.svg`) © Clube Zona Sul, all rights reserved, not covered by the GPL.
Fonts (`public/assets/fonts/`): [Outfit](https://github.com/Outfitio/Outfit-Fonts), standing in for the club's Codec Pro, and [Shippori Mincho](https://github.com/fontdasu/ShipporiMincho) ExtraBold cut down to the ten kanji/kana the pages show. Both SIL Open Font License 1.1.
Tile art: [FluffyStuff/riichi-mahjong-tiles](https://github.com/FluffyStuff/riichi-mahjong-tiles), public domain (CC0).
