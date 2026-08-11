import re

with open("frontend/src/components/GuInventory.tsx", "r", encoding="utf-8") as f:
    content = f.read()

# 1. Update imports
content = content.replace(
    'import { Flower2, Wine, Gem, ShieldAlert, Sparkles, HelpCircle, Search, Shield, Zap, ChevronDown, ChevronUp, Lock, Unlock } from "lucide-react";',
    'import { Flower2, Wine, Gem, ShieldAlert, Sparkles, HelpCircle, Search, Shield, Zap, ChevronDown, ChevronUp, Lock, Unlock, ChevronRight, X, ArrowLeft, ArrowDownAZ, SortDesc, Filter } from "lucide-react";'
)

# 2. Update states
state_old = """  const [feedingId, setFeedingId] = useState<string | null>(null);
  const [feedingGroupGuId, setFeedingGroupGuId] = useState<string | null>(null);
  const [expandedGroups, setExpandedGroups] = useState<Record<string, boolean>>({});
  const [equippingId, setEquippingId] = useState<string | null>(null);
  const [usingId, setUsingId] = useState<string | null>(null);
  const [statusUpdatingId, setStatusUpdatingId] = useState<string | null>(null);
  const [usingGuId, setUsingGuId] = useState<string | null>(null);
  const [settingVitalId, setSettingVitalId] = useState<string | null>(null);
  const [droppingId, setDroppingId] = useState<string | null>(null);
  const [detailOpen, setDetailOpen] = useState<Record<string, boolean>>({});
  const [itemDetailOpen, setItemDetailOpen] = useState<Record<string, boolean>>({});
  const [advancedOpen, setAdvancedOpen] = useState<Record<string, boolean>>({});"""

state_new = """  const [feedingId, setFeedingId] = useState<string | null>(null);
  const [feedingGroupGuId, setFeedingGroupGuId] = useState<string | null>(null);
  const [equippingId, setEquippingId] = useState<string | null>(null);
  const [usingId, setUsingId] = useState<string | null>(null);
  const [statusUpdatingId, setStatusUpdatingId] = useState<string | null>(null);
  const [usingGuId, setUsingGuId] = useState<string | null>(null);
  const [settingVitalId, setSettingVitalId] = useState<string | null>(null);
  const [droppingId, setDroppingId] = useState<string | null>(null);
  const [itemDetailOpen, setItemDetailOpen] = useState<Record<string, boolean>>({});

  // Drawer States
  const [activeGroupKey, setActiveGroupKey] = useState<string | null>(null);
  const [activeGroupViewMode, setActiveGroupViewMode] = useState<"summary" | "list">("summary");
  const [activeInstanceId, setActiveInstanceId] = useState<string | null>(null);
  const [groupListTab, setGroupListTab] = useState<"all" | "equipped" | "feeding" | "idle">("all");
  const [groupListSearch, setGroupListSearch] = useState("");
  const [groupListSort, setGroupListSort] = useState<"hunger_asc" | "hunger_desc">("hunger_desc");
  const [groupFeedAmount, setGroupFeedAmount] = useState<number>(10);"""

content = content.replace(state_old, state_new)

# Remove toggleGroupExpand
toggle_expand_old = """  const toggleGroupExpand = (guId: string) => {
    setExpandedGroups((prev) => ({ ...prev, [guId]: !prev[guId] }));
  };"""
content = content.replace(toggle_expand_old, "")

with open("frontend/src/components/GuInventory.tsx", "w", encoding="utf-8") as f:
    f.write(content)
