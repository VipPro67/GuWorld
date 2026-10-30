import assert from "node:assert/strict";
import test from "node:test";
import { CastController, type CastDelegate } from "./CastController.ts";
import type { RuntimeEntity } from "../core/WorldRuntime.ts";
import type { WorldSkillInfo } from "../network/Protocol.ts";
import type { WorldPoint } from "../world/WorldUnits.ts";

function skill(overrides: Partial<WorldSkillInfo> = {}): WorldSkillInfo {
  return {
    slot: 1,
    id: "gu-1",
    name: "Cổ thử",
    mode: "SELF",
    mana_cost: 10,
    range: 5,
    cooldown_ticks: 0,
    windup_ticks: 0,
    ...overrides,
  };
}

function entity(overrides: Partial<RuntimeEntity> = {}): RuntimeEntity {
  return {
    id: "m1",
    kind: "monster",
    partyId: "",
    x: 4,
    y: 0,
    hp: 10,
    maxHp: 10,
    shield: 0,
    templateId: "monster_sha",
    name: "Quái",
    aiState: "idle",
    aggroTargetId: "",
    isSelf: false,
    ...overrides,
  };
}

class RecordingView {
  public draws = 0;
  public previews = 0;
  public clears = 0;
  public drawAim(): void {
    this.draws += 1;
  }
  public drawPreview(): void {
    this.previews += 1;
  }
  public clear(): void {
    this.clears += 1;
  }
}

interface Harness {
  controller: CastController;
  intents: Array<Record<string, unknown>>;
  notices: string[];
  marks: WorldPoint[];
  selections: (string | null)[];
  aiming: (number | null)[];
  view: RecordingView;
}

function makeHarness(options: {
  skills?: WorldSkillInfo[];
  entities?: RuntimeEntity[];
  selected?: string | null;
  pointer?: WorldPoint;
  joined?: boolean;
} = {}): Harness {
  let selected = options.selected ?? null;
  const pointer = options.pointer ?? { x: 2, y: 0 };
  const recorder: Harness = {
    intents: [],
    notices: [],
    marks: [],
    selections: [],
    aiming: [],
    view: new RecordingView(),
    controller: null as unknown as CastController,
  };
  const delegate: CastDelegate = {
    isJoined: () => options.joined ?? true,
    skillForSlot: (slot) => (options.skills ?? []).find((entry) => entry.slot === slot) ?? null,
    selfPosition: () => ({ x: 0, y: 0 }),
    renderedSelfPosition: () => ({ x: 0, y: 0 }),
    pointerWorld: () => pointer,
    selectedTargetId: () => selected,
    selectTarget: (entityId) => {
      selected = entityId;
      recorder.selections.push(entityId);
    },
    setAiming: (slot) => recorder.aiming.push(slot),
    attackableEntities: () => options.entities ?? [],
    entityNear: () => (options.entities ?? []).find(
      (candidate) => Math.hypot(candidate.x - pointer.x, candidate.y - pointer.y) <= 1.35,
    ) ?? null,
    notice: (message) => recorder.notices.push(message),
    markPoint: (point) => recorder.marks.push({ ...point }),
    sendCast: (intent) => recorder.intents.push(intent),
    view: recorder.view,
  };
  recorder.controller = new CastController(delegate);
  return recorder;
}

test("SELF skills cast immediately without an aim phase", () => {
  const { controller, intents } = makeHarness({ skills: [skill({ mode: "SELF" })] });
  controller.triggerSlot(1);
  assert.deepEqual(intents, [{ slot: 1, mode: "SELF" }]);
  assert.equal(controller.isAiming(), false);
});

test("DIRECTION skills enter aim on first press and commit on pointer", () => {
  const { controller, intents, aiming, marks } = makeHarness({
    skills: [skill({ mode: "DIRECTION" })],
    pointer: { x: 2, y: 0 },
  });
  controller.triggerSlot(1);
  assert.equal(controller.aimingSlot(), 1);
  assert.deepEqual(aiming, [1]);
  assert.equal(intents.length, 0);

  controller.triggerSlot(1, { fromPointer: true });
  assert.deepEqual(intents, [{ slot: 1, mode: "DIRECTION", aim_x: 1, aim_y: 0 }]);
  assert.deepEqual(marks, [{ x: 2, y: 0 }]);
  assert.equal(controller.isAiming(), false);
});

test("TARGET skills with no selection enter aim and announce it", () => {
  const { controller, intents, aiming, notices } = makeHarness({
    skills: [skill({ mode: "TARGET" })],
  });
  controller.triggerSlot(1);
  assert.equal(controller.aimingSlot(), 1);
  assert.deepEqual(aiming, [1]);
  assert.equal(intents.length, 0);
  assert.equal(notices.length, 1);
  assert.match(notices[0], /nhấp vào mục tiêu/);

  controller.cancelAim();
  assert.equal(controller.aimingSlot(), null);
  assert.deepEqual(aiming, [1, null]);
});

test("TARGET commits at the selected entity for keyboard activation", () => {
  const { controller, intents, selections, marks } = makeHarness({
    skills: [skill({ mode: "TARGET" })],
    entities: [entity({ id: "m1" })],
    selected: "m1",
  });
  controller.triggerSlot(1);
  assert.deepEqual(intents, [{ slot: 1, mode: "TARGET", target_id: "m1" }]);
  assert.deepEqual(selections, ["m1"]);
  assert.deepEqual(marks, [{ x: 4, y: 0 }]);
});

test("TARGET commits at an entity near the pointer click", () => {
  const { controller, intents, selections } = makeHarness({
    skills: [skill({ mode: "TARGET" })],
    entities: [entity({ id: "m2", x: 4, y: 0 })],
    pointer: { x: 4, y: 0 },
  });
  controller.triggerSlot(1, { fromPointer: true });
  assert.deepEqual(intents, [{ slot: 1, mode: "TARGET", target_id: "m2" }]);
  assert.deepEqual(selections, ["m2"]);
});

test("TARGET falls back to a directional aim intent on empty ground", () => {
  const { controller, intents, marks } = makeHarness({
    skills: [skill({ mode: "TARGET" })],
    pointer: { x: 8, y: 0 },
  });
  controller.triggerSlot(1, { fromPointer: true });
  assert.deepEqual(intents, [{ slot: 1, mode: "TARGET", aim_x: 8, aim_y: 0 }]);
  assert.deepEqual(marks, [{ x: 8, y: 0 }]);
});

test("GROUND skills enter aim on keyboard press and commit on pointer", () => {
  const { controller, intents, marks } = makeHarness({
    skills: [skill({ mode: "GROUND" })],
    pointer: { x: 3, y: -1 },
  });
  controller.triggerSlot(1);
  assert.equal(controller.aimingSlot(), 1);
  assert.equal(intents.length, 0);

  controller.triggerSlot(1, { fromPointer: true });
  assert.deepEqual(intents, [{ slot: 1, mode: "GROUND", x: 3, y: -1 }]);
  assert.deepEqual(marks, [{ x: 3, y: -1 }]);
});

test("unassigned slots and offline sessions notify instead of casting", () => {
  const { controller, intents, notices } = makeHarness({ skills: [] });
  controller.triggerSlot(3);
  assert.equal(intents.length, 0);
  assert.equal(notices.at(-1), "Nút 3 chưa có Cổ hỗ trợ realtime");

  const offline = makeHarness({ skills: [skill()], joined: false });
  offline.controller.triggerSlot(1);
  assert.equal(offline.intents.length, 0);
  assert.equal(offline.notices.at(-1), "Chưa kết nối world server");
});

test("castAtEntity sends a direct target_id intent for auto-farm", () => {
  const { controller, intents } = makeHarness({
    skills: [skill({ mode: "TARGET", slot: 2 })],
  });
  controller.castAtEntity(2, entity({ id: "boss" }));
  assert.deepEqual(intents, [{ slot: 2, mode: "TARGET", target_id: "boss" }]);
});

test("update drives the telegraph view from aim and hover state", () => {
  const { controller, view } = makeHarness({
    skills: [skill({ mode: "DIRECTION" })],
  });
  controller.update();
  assert.equal(view.clears, 1);

  controller.setHoveredSlot(null);
  assert.equal(view.clears, 2);

  controller.triggerSlot(1); // enters aim
  controller.update();
  assert.equal(view.draws, 1);

  controller.setHoveredSlot(1); // ignored while aiming
  controller.update();
  assert.equal(view.draws, 2);

  controller.cancelAim();
  assert.equal(view.clears, 3);

  controller.update();
  assert.equal(view.previews, 1);

  controller.setHoveredSlot(null);
  assert.equal(view.clears, 4);

  controller.update();
  assert.equal(view.clears, 5);
});
