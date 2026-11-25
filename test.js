const assert = require('assert');
const c = require('./public/common.js');

assert.deepStrictEqual(c.seatWinds(1), ['東', '南', '西', '北']);
assert.deepStrictEqual(c.seatWinds(2), ['北', '東', '南', '西']);
assert.deepStrictEqual(c.seatWinds(4), ['南', '西', '北', '東']);

const s = sc => sc.map(score => ({ score }));
assert.deepStrictEqual(c.placements(s([25000, 32000, 18000, 25000])), [2, 1, 4, 3]);

assert.deepStrictEqual(c.parsePlayers('Ana Souza 45,5\n\n  Bob -12.3 \nPedro'),
  [{ name: 'Ana Souza', total: 45.5 }, { name: 'Bob', total: -12.3 }, { name: 'Pedro', total: 0 }]);
const ps = [{ name: 'Player 2', total: 0 }];
assert.deepStrictEqual(c.parsePlayers(c.playersText(ps)), ps);

assert.deepStrictEqual(c.nextHand({ wind: 0, num: 4, honba: 3, riichi: 1 }), { wind: 1, num: 1, honba: 0, riichi: 1 });
assert.deepStrictEqual(c.nextHand({ wind: 0, num: 2, honba: 1, riichi: 0 }), { wind: 0, num: 3, honba: 0, riichi: 0 });

const g = { round: { riichi: 0 }, seats: [{ score: 25000 }] };
c.toggleRiichi(g, 0);
assert.deepStrictEqual(g, { round: { riichi: 1 }, seats: [{ score: 24000, riichi: true }] });
g.round.riichi = 0; // sticks already paid out to a winner
c.toggleRiichi(g, 0);
assert.deepStrictEqual(g, { round: { riichi: 0 }, seats: [{ score: 25000, riichi: false }] });

// Win payouts. Hand 東2: seat 1 is dealer.
const w = (honba, riichi) => ({ round: { num: 2, honba, riichi } });
assert.deepStrictEqual(c.winDeltas(w(2, 1), 0, 3, 8000), [9600, 0, 0, -8600]);   // ron + 2 honba + 1 stick
assert.deepStrictEqual(c.winDeltas(w(1, 0), 0, null, 8000), [8300, -4100, -2100, -2100]); // non-dealer tsumo mangan
assert.deepStrictEqual(c.winDeltas(w(0, 2), 1, null, 3900), [-1300, 5900, -1300, -1300]); // dealer tsumo 1300 all + 2 sticks
assert.deepStrictEqual(c.winDeltas(w(0, 0), 2, null, 3900), [-1000, -2000, 4000, -1000]); // 30fu 3han tsumo = 1000/2000

// Double ron: seat 0 deals in to 3 and 1; seat 1 is first after the loser, so it takes honba + sticks.
assert.deepStrictEqual(c.ronDeltas(w(1, 2), 0, [[3, 8000], [1, 3900]]), [-12200, 6200, 0, 8000]);
assert.deepStrictEqual(c.ronDeltas(w(2, 1), 0, [[3, 8000]]), c.winDeltas(w(2, 1), 3, 0, 8000)); // single ron unchanged

// Splitting the ron value matches the real fu/han tsumo table for every non-limit hand.
for (const fu of [20, 25, 30, 40, 50, 60, 70, 80, 90, 100, 110]) for (let han = 1; han <= 4; han++) {
  const b = fu * 2 ** (han + 2);
  if (b >= 2000) continue; // mangan and up: presets
  assert.strictEqual(c.ceil100(c.ceil100(4 * b) / 4), c.ceil100(b), `${fu}fu ${han}han`);
  assert.strictEqual(c.ceil100(c.ceil100(4 * b) / 2), c.ceil100(2 * b), `${fu}fu ${han}han`);
  assert.strictEqual(c.ceil100(c.ceil100(6 * b) / 3), c.ceil100(2 * b), `${fu}fu ${han}han dealer`);
}

// Exhaustive draw tenpai payments.
assert.deepStrictEqual(c.drawDeltas([true, false, false, false]), [3000, -1000, -1000, -1000]);
assert.deepStrictEqual(c.drawDeltas([true, true, false, false]), [1500, 1500, -1500, -1500]);
assert.deepStrictEqual(c.drawDeltas([true, true, true, false]), [1000, 1000, 1000, -3000]);
assert.deepStrictEqual(c.drawDeltas([false, false, false, false]), [0, 0, 0, 0]);
assert.deepStrictEqual(c.drawDeltas([true, true, true, true]), [0, 0, 0, 0]);

// Wait tiles: 34 kinds, each with an image file.
assert.strictEqual(c.TILES.length, 34);
for (const t of c.TILES) assert.ok(require('fs').existsSync(`${__dirname}/public/${c.tileSrc(t)}`), t);

console.log('all tests passed');
