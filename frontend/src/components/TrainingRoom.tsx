import React, { useState, useEffect } from 'react';
import { useGameStore, CombatState } from '../store/useGameStore';
import { CombatScreen } from './CombatScreen';

interface TrainingRoomProps {
  onClose: () => void;
}

export const TrainingRoom: React.FC<TrainingRoomProps> = ({ onClose }) => {
  const store = useGameStore();

  const [dummyConfig, setDummyConfig] = useState({
    dummyHP: 100000,
    dummyShield: 0,
    dummyAtk: 1000,
    dummyDef: 500,
    dummySpeed: 1000
  });

  useEffect(() => {
    // Load saved settings
    const saved = localStorage.getItem('guworld_training_dummy');
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        setDummyConfig(prev => ({
          ...prev,
          ...parsed
        }));
      } catch (e) {
        console.error("Failed to parse saved dummy settings");
      }
    }
  }, []);

  const handleConfigChange = (newConfig: any) => {
    setDummyConfig(newConfig);
    localStorage.setItem('guworld_training_dummy', JSON.stringify(newConfig));
  };

  if (!store.player) return null;

  // Create a mock CombatState for the dummy so CombatScreen doesn't throw errors
  const mockDummyState: CombatState = {
    enemy_name: "Hình Nhân Gỗ",
    enemy_hp: dummyConfig.dummyHP,
    enemy_max_hp: dummyConfig.dummyHP,
    enemy_damage: dummyConfig.dummyAtk,
    enemy_defense: dummyConfig.dummyDef,
    charge_points: 0,
    enemy_type: "Enemy",
    enemy_rank: 1,
    enemy_sub_rank: 1,
    combat_mode: "practice",
    enemies: [
      {
        id: "practice_dummy",
        name: "Hình Nhân Gỗ",
        kind: "Enemy",
        rank: 1,
        sub_rank: 1,
        hp: dummyConfig.dummyHP,
        max_hp: dummyConfig.dummyHP,
        current_mana: 0,
        max_mana: 0,
        base_attack: dummyConfig.dummyAtk,
        base_defense: dummyConfig.dummyDef,
        speed: dummyConfig.dummySpeed,
        grid_x: 1,
        grid_y: 2
      }
    ],
    allies: [
      {
        id: store.player.id,
        name: store.player.username,
        kind: "Player",
        rank: store.player.rank,
        sub_rank: store.player.sub_rank,
        hp: store.player.base_hp,
        max_hp: store.player.base_hp,
        current_mana: store.player.current_mana,
        max_mana: store.player.max_mana,
        base_attack: 0,
        base_defense: 0,
        speed: 0,
        grid_x: 2,
        grid_y: 2
      }
    ]
  };

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center bg-black/80 backdrop-blur-sm overflow-y-auto">
      <div className="relative w-full py-10 px-4">
        {/* We reuse CombatScreen with isPracticeMode enabled */}
        <CombatScreen 
          player={store.player}
          enemy={mockDummyState}
          guDictionary={store.guDictionary}
          actionLoading={store.actionLoading}
          onRetreat={async () => { onClose(); }}
          isPracticeMode={true}
          practiceDummyConfig={dummyConfig}
          onPracticeDummyConfigChange={handleConfigChange}
        />
      </div>
    </div>
  );
};
