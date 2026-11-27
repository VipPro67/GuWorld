// THROWAWAY PROTOTYPE DATA for Wayfinder #480 — Aperture Variant Labor & Immortals

export type RaceKey =
  | 'HAIRY_MAN'
  | 'ROCKMAN'
  | 'FEATHERMAN'
  | 'SNOWMAN'
  | 'INKMAN'
  | 'MINIMAN'
  | 'MERMAN'
  | 'MUSHROOMMAN'
  | 'DRAGONMAN';

export type LaborIntensity = 'LOW' | 'NORMAL' | 'HIGH' | 'EXPLOIT';
export type SettlementType = 'VILLAGE' | 'TOWN' | 'CITY' | 'FORTRESS';
export type NPCRole = 'NONE' | 'REFINER' | 'MINER' | 'FARMER' | 'STEWARD';
export type EventSeverity = 'STRIKE' | 'SABOTAGE' | 'REBELLION';

export interface RaceState {
  race: RaceKey;
  name: string;
  specialty: string;
  daoPath: string;
  daoMarksCurrent: number;
  daoMarksRequired: number;
  population: number;
  capacity: number;
  settlementType: SettlementType | null;
  settlementPos: { x: number; y: number } | null;
  happiness: number;
  oppression: number;
  rebellionRate: number;
  laborIntensity: LaborIntensity;
  foodStored: number;
  foodStorageCap: number;
  autoEssenceRelief: boolean;
  starving: boolean;
  guMasterCount: number;
  guImmortalCount: number;
  reproRateLabel: string;
  immortalRateLabel: string;
}

export interface ResourcePointMock {
  id: string;
  name: string;
  pointType: string;
  path: string;
  workerRace: RaceKey;
  assignedRace: RaceKey | null;
  assignedPop: number;
  vitality: number;
  isFood: boolean;
  pos: { x: number; y: number };
  baseYield: number;
}

export interface VariantEventMock {
  id: string;
  race: RaceKey;
  eventType: EventSeverity;
  description: string;
  resolved: boolean;
  createdAt: string;
}

export interface VariantNPCMock {
  id: string;
  name: string;
  race: RaceKey;
  rank: number; // 1..5 mortal, 6..8 immortal
  subRank: number;
  path: string;
  role: NPCRole;
  loyalty: number;
  talent: number;
  formationSlot: number; // 0 = not in formation
  guEquipped: string[];
  daoMarks: number;
  isImmortal: boolean;
  defenseStationed: boolean;
}

export const INITIAL_RACES: RaceState[] = [
  {
    race: 'HAIRY_MAN',
    name: 'Mao Dân',
    specialty: 'Luyện Đạo · Luyện Cổ Tự Động',
    daoPath: 'Refinement Path',
    daoMarksCurrent: 4200,
    daoMarksRequired: 2000,
    population: 18500,
    capacity: 50000,
    settlementType: 'TOWN',
    settlementPos: { x: 1, y: 1 },
    happiness: 72,
    oppression: 24,
    rebellionRate: 14,
    laborIntensity: 'HIGH',
    foodStored: 12400,
    foodStorageCap: 25000,
    autoEssenceRelief: false,
    starving: false,
    guMasterCount: 18,
    guImmortalCount: 1,
    reproRateLabel: 'Nhanh',
    immortalRateLabel: 'Thấp',
  },
  {
    race: 'ROCKMAN',
    name: 'Thạch Nhân',
    specialty: 'Thổ/Kim · Khai Khoáng',
    daoPath: 'Earth Path',
    daoMarksCurrent: 3100,
    daoMarksRequired: 2000,
    population: 14200,
    capacity: 50000,
    settlementType: 'TOWN',
    settlementPos: { x: 3, y: 2 },
    happiness: 38,
    oppression: 62,
    rebellionRate: 54,
    laborIntensity: 'EXPLOIT',
    foodStored: 8000,
    foodStorageCap: 20000,
    autoEssenceRelief: false,
    starving: false,
    guMasterCount: 11,
    guImmortalCount: 0,
    reproRateLabel: 'Trung bình',
    immortalRateLabel: 'Trung bình',
  },
  {
    race: 'FEATHERMAN',
    name: 'Vũ Dân',
    specialty: 'Phong/Không · Tiết Kiệm Khí',
    daoPath: 'Wind Path',
    daoMarksCurrent: 1200,
    daoMarksRequired: 1800,
    population: 4800,
    capacity: 10000,
    settlementType: 'VILLAGE',
    settlementPos: { x: 0, y: 3 },
    happiness: 64,
    oppression: 10,
    rebellionRate: 6,
    laborIntensity: 'NORMAL',
    foodStored: 3500,
    foodStorageCap: 10000,
    autoEssenceRelief: false,
    starving: false,
    guMasterCount: 6,
    guImmortalCount: 0,
    reproRateLabel: 'Trung bình',
    immortalRateLabel: 'Cao',
  },
  {
    race: 'MINIMAN',
    name: 'Tiểu Nhân',
    specialty: 'Mộc/Thực · Canh Tác Lương Thực',
    daoPath: 'Wood Path',
    daoMarksCurrent: 2800,
    daoMarksRequired: 1500,
    population: 9200,
    capacity: 10000,
    settlementType: 'VILLAGE',
    settlementPos: { x: 2, y: 0 },
    happiness: 86,
    oppression: 4,
    rebellionRate: 2,
    laborIntensity: 'NORMAL',
    foodStored: 9800,
    foodStorageCap: 10000,
    autoEssenceRelief: false,
    starving: false,
    guMasterCount: 5,
    guImmortalCount: 0,
    reproRateLabel: 'Rất nhanh',
    immortalRateLabel: 'Thấp',
  },
  {
    race: 'MERMAN',
    name: 'Giao Nhân',
    specialty: 'Thủy/Huyết · Thủy Sản Long Ngư',
    daoPath: 'Water Path',
    daoMarksCurrent: 2100,
    daoMarksRequired: 2000,
    population: 6100,
    capacity: 10000,
    settlementType: 'VILLAGE',
    settlementPos: { x: 4, y: 4 },
    happiness: 24,
    oppression: 45,
    rebellionRate: 71,
    laborIntensity: 'HIGH',
    foodStored: 0,
    foodStorageCap: 10000,
    autoEssenceRelief: true,
    starving: true,
    guMasterCount: 7,
    guImmortalCount: 0,
    reproRateLabel: 'Trung bình',
    immortalRateLabel: 'Trung bình',
  },
  {
    race: 'SNOWMAN',
    name: 'Tuyết Nhân',
    specialty: 'Băng/Tuyết · Tăng Tỉ Lệ Hiếm',
    daoPath: 'Ice Path',
    daoMarksCurrent: 900,
    daoMarksRequired: 2500,
    population: 1200,
    capacity: 10000,
    settlementType: 'VILLAGE',
    settlementPos: { x: 4, y: 0 },
    happiness: 55,
    oppression: 12,
    rebellionRate: 8,
    laborIntensity: 'LOW',
    foodStored: 2400,
    foodStorageCap: 10000,
    autoEssenceRelief: false,
    starving: false,
    guMasterCount: 2,
    guImmortalCount: 0,
    reproRateLabel: 'Chậm',
    immortalRateLabel: 'Cao',
  },
  {
    race: 'INKMAN',
    name: 'Mặc Nhân',
    specialty: 'Trí/Luật · Quản Trị & Nghiên Cứu',
    daoPath: 'Wisdom Path',
    daoMarksCurrent: 1600,
    daoMarksRequired: 2200,
    population: 0,
    capacity: 0,
    settlementType: null,
    settlementPos: null,
    happiness: 50,
    oppression: 0,
    rebellionRate: 0,
    laborIntensity: 'NORMAL',
    foodStored: 0,
    foodStorageCap: 0,
    autoEssenceRelief: false,
    starving: false,
    guMasterCount: 0,
    guImmortalCount: 0,
    reproRateLabel: 'Chậm',
    immortalRateLabel: 'Cao',
  },
  {
    race: 'MUSHROOMMAN',
    name: 'Nấm Nhân',
    specialty: 'Độc/Hắc Ám · Miễn Nhiễm Độc Địa',
    daoPath: 'Poison Path',
    daoMarksCurrent: 600,
    daoMarksRequired: 1800,
    population: 0,
    capacity: 0,
    settlementType: null,
    settlementPos: null,
    happiness: 50,
    oppression: 0,
    rebellionRate: 0,
    laborIntensity: 'NORMAL',
    foodStored: 0,
    foodStorageCap: 0,
    autoEssenceRelief: false,
    starving: false,
    guMasterCount: 0,
    guImmortalCount: 0,
    reproRateLabel: 'Nhanh',
    immortalRateLabel: 'Thấp',
  },
  {
    race: 'DRAGONMAN',
    name: 'Long Nhân',
    specialty: 'Biến Hóa/Nô · Chiến Đấu & Thuần Thú',
    daoPath: 'Transformation Path',
    daoMarksCurrent: 3500,
    daoMarksRequired: 3000,
    population: 3400,
    capacity: 10000,
    settlementType: 'VILLAGE',
    settlementPos: { x: 2, y: 3 },
    happiness: 78,
    oppression: 15,
    rebellionRate: 9,
    laborIntensity: 'NORMAL',
    foodStored: 4100,
    foodStorageCap: 10000,
    autoEssenceRelief: false,
    starving: false,
    guMasterCount: 9,
    guImmortalCount: 1,
    reproRateLabel: 'Chậm',
    immortalRateLabel: 'Rất cao',
  },
];

export const INITIAL_POINTS: ResourcePointMock[] = [
  {
    id: 'pt-1',
    name: 'Luyện Thiết Lò',
    pointType: 'REFINEMENT_IRON_MINE',
    path: 'Refinement Path',
    workerRace: 'HAIRY_MAN',
    assignedRace: 'HAIRY_MAN',
    assignedPop: 500,
    vitality: 88,
    isFood: false,
    pos: { x: 1, y: 2 },
    baseYield: 10,
  },
  {
    id: 'pt-2',
    name: 'Mạch Quặng Địa Mạch',
    pointType: 'ORE_VEIN',
    path: 'Earth Path',
    workerRace: 'ROCKMAN',
    assignedRace: 'ROCKMAN',
    assignedPop: 500,
    vitality: 52,
    isFood: false,
    pos: { x: 3, y: 1 },
    baseYield: 10,
  },
  {
    id: 'pt-3',
    name: 'Đồng Cỏ Dạ Thú',
    pointType: 'BEAST_PASTURE',
    path: 'Food Path',
    workerRace: 'MINIMAN',
    assignedRace: 'MINIMAN',
    assignedPop: 400,
    vitality: 95,
    isFood: true,
    pos: { x: 2, y: 1 },
    baseYield: 12,
  },
  {
    id: 'pt-4',
    name: 'Ao Linh Tuyền',
    pointType: 'SPIRIT_SPRING',
    path: 'Water Path',
    workerRace: 'MERMAN',
    assignedRace: 'HAIRY_MAN',
    assignedPop: 300,
    vitality: 76,
    isFood: false,
    pos: { x: 3, y: 4 },
    baseYield: 10,
  },
  {
    id: 'pt-5',
    name: 'Núi Băng Thủy',
    pointType: 'FROSTY_PEAK',
    path: 'Ice Path',
    workerRace: 'SNOWMAN',
    assignedRace: 'SNOWMAN',
    assignedPop: 350,
    vitality: 90,
    isFood: false,
    pos: { x: 4, y: 1 },
    baseYield: 10,
  },
];

export const INITIAL_EVENTS: VariantEventMock[] = [
  {
    id: 'ev-1',
    race: 'ROCKMAN',
    eventType: 'STRIKE',
    description: 'Thạch Nhân đình công phản đối cường độ Bóc Lột tại Mạch Quặng Địa Mạch (đóng băng bonus lao động).',
    resolved: false,
    createdAt: '12 phút trước',
  },
  {
    id: 'ev-2',
    race: 'MERMAN',
    eventType: 'SABOTAGE',
    description: 'Giao Nhân thiếu lương thực lâu ngày gây phá hoại Ao Linh Tuyền (-15% sinh khí điểm tài nguyên).',
    resolved: false,
    createdAt: '4 phút trước',
  },
];

export const INITIAL_NPCS: VariantNPCMock[] = [
  {
    id: 'npc-1',
    name: 'Mao Lục Trưởng Lão',
    race: 'HAIRY_MAN',
    rank: 6,
    subRank: 1,
    path: 'Refinement Path',
    role: 'REFINER',
    loyalty: 92,
    talent: 88,
    formationSlot: 0,
    guEquipped: ['Tiên Cổ: Luyện Ý (R6)', 'Hỏa Lô Cổ (R5)'],
    daoMarks: 2400,
    isImmortal: true,
    defenseStationed: true,
  },
  {
    id: 'npc-2',
    name: 'Long Chiến Tiên Tôn',
    race: 'DRAGONMAN',
    rank: 7,
    subRank: 2,
    path: 'Transformation Path',
    role: 'NONE',
    loyalty: 95,
    talent: 94,
    formationSlot: 1,
    guEquipped: ['Tiên Cổ: Long Lân (R7)', 'Long Khiếu Cổ (R6)'],
    daoMarks: 8600,
    isImmortal: true,
    defenseStationed: true,
  },
  {
    id: 'npc-3',
    name: 'Thạch Khôi',
    race: 'ROCKMAN',
    rank: 5,
    subRank: 4,
    path: 'Earth Path',
    role: 'MINER',
    loyalty: 84,
    talent: 79,
    formationSlot: 0,
    guEquipped: ['Thổ Bích Cổ (R5)'],
    daoMarks: 0,
    isImmortal: false,
    defenseStationed: false,
  },
  {
    id: 'npc-4',
    name: 'Tiểu Mộc',
    race: 'MINIMAN',
    rank: 4,
    subRank: 2,
    path: 'Wood Path',
    role: 'FARMER',
    loyalty: 90,
    talent: 66,
    formationSlot: 0,
    guEquipped: ['Sinh Cơ Thảo Cổ (R4)'],
    daoMarks: 0,
    isImmortal: false,
    defenseStationed: false,
  },
];
