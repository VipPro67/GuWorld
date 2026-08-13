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
    id: "practice_dummy",
    status: "ACTIVE",
    enemy_template_id: "",
    enemy_max_hp: dummyConfig.dummyHP,
    boss_phase: 0,
    current_turn: 0,
    logs: [],
    enemies: [
      {
        id: "practice_dummy",
        name: "Hình Nhân Gỗ",
        kind: "Enemy",
        rank: 1,
        sub_rank: 1,
        max_hp: dummyConfig.dummyHP,
        hp: dummyConfig.dummyHP,
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
        max_hp: 0,
        hp: 0,
        base_attack: 0,
        base_defense: 0,
        speed: 0,
        grid_x: 2,
        grid_y: 2
      }
    ]
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm overflow-y-auto">
      <div className="relative w-full max-w-4xl min-h-screen py-10 px-4">
        {/* We reuse CombatScreen with isPracticeMode enabled */}
        <CombatScreen 
          player={store.player}
          enemy={mockDummyState}
          guDictionary={store.guDictionary}
          actionLoading={store.actionLoading}
          onRetreat={onClose}
          isPracticeMode={true}
          practiceDummyConfig={dummyConfig}
          onPracticeDummyConfigChange={handleConfigChange}
        />
      </div>
    </div>
  );
};
