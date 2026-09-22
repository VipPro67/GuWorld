import assert from "node:assert/strict";
import test from "node:test";
import { syncRecipes } from "./recipeSync.ts";
import { fetchWithDedupe } from "../../lib/fetch.ts";

const response = (status, data, etag) => new Response(
  data === undefined ? null : JSON.stringify(data),
  { status, headers: etag ? { ETag: etag } : undefined },
);

test("an initial recipe failure remains retryable and a later valid response loads", async () => {
  let state = { recipes: [], recipesLoaded: false, recipesLoading: false, recipesError: null, recipesFetchedAt: null };
  let calls = 0;
  const run = () => syncRecipes({
    cacheKey: "game_recipes_player-1",
    initialLoaded: state.recipesLoaded,
    readCache: async () => null,
    writeCache: async () => {},
    request: async () => (++calls === 1 ? response(500) : response(200, [{ id: "recipe_1" }], '"v2"')),
    setState: (patch) => { state = { ...state, ...patch }; },
    now: () => 1234,
  });

  await run();
  assert.equal(state.recipesLoaded, false);
  assert.equal(state.recipesLoading, false);
  assert.match(state.recipesError, /500/);
  assert.equal(state.recipesFetchedAt, null);

  await run();
  assert.equal(calls, 2);
  assert.equal(state.recipesLoaded, true);
  assert.equal(state.recipesError, null);
  assert.equal(state.recipesFetchedAt, 1234);
  assert.deepEqual(state.recipes, [{ id: "recipe_1" }]);
});

test("304 keeps the hydrated recipe cache", async () => {
  const cachedRecipes = [{ id: "recipe_cached" }];
  let state = { recipes: [], recipesLoaded: false };
  let sentETag;

  await syncRecipes({
    cacheKey: "game_recipes_player-1",
    initialLoaded: false,
    readCache: async () => ({ data: cachedRecipes, etag: '"cached"', timestamp: 100 }),
    writeCache: async () => assert.fail("304 must not replace the cache"),
    request: async (etag) => {
      sentETag = etag;
      return response(304);
    },
    setState: (patch) => { state = { ...state, ...patch }; },
    now: () => 200,
  });

  assert.equal(sentETag, '"cached"');
  assert.strictEqual(state.recipes, cachedRecipes);
  assert.equal(state.recipesLoaded, true);
  assert.equal(state.recipesError, null);
  assert.equal(state.recipesFetchedAt, 200);
});

test("concurrent recipe requests are deduplicated", async () => {
  let release;
  let calls = 0;
  const pending = new Promise((resolve) => { release = resolve; });
  const fetcher = async () => { calls += 1; await pending; };

  const first = fetchWithDedupe("recipes:test", fetcher);
  const second = fetchWithDedupe("recipes:test", fetcher);
  assert.equal(calls, 1);
  release();
  await Promise.all([first, second]);
  assert.equal(calls, 1);
});
