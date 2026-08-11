import re

with open("frontend/src/components/GuInventory.tsx", "r", encoding="utf-8") as f:
    content = f.read()

# 1. Add createPortal import
content = content.replace(
    'import React, { useEffect, useState } from "react";\n',
    'import React, { useEffect, useState } from "react";\nimport { createPortal } from "react-dom";\n'
)

# 2. Add mounted state at the beginning of GuInventory component
# Find the start of the component body
comp_start = content.find("export const GuInventory: React.FC<GuInventoryProps> = ({")
# Find the end of the destructuring
body_start = content.find(") => {\n", comp_start) + 7

mounted_code = """  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);

"""
content = content[:body_start] + mounted_code + content[body_start:]

# 3. Update renderGroupDrawer
old_group_return = """    return (
      <div className="fixed inset-0 z-50 flex flex-col justify-end bg-black/80 animate-in fade-in">"""
new_group_return = """    if (!mounted) return null;
    return createPortal(
      <div className="fixed inset-0 z-50 flex flex-col justify-end bg-black/80 animate-in fade-in">"""

content = content.replace(old_group_return, new_group_return)

# Close createPortal for Group Drawer
old_group_end = """        </div>
      </div>
    );
  };

  const renderSingleInstanceDrawer"""
new_group_end = """        </div>
      </div>,
      document.body
    );
  };

  const renderSingleInstanceDrawer"""

content = content.replace(old_group_end, new_group_end)

# 4. Update renderSingleInstanceDrawer
old_single_return = """    return (
      <div className="fixed inset-0 z-[60] flex flex-col justify-end bg-black/80 animate-in fade-in">"""
new_single_return = """    if (!mounted) return null;
    return createPortal(
      <div className="fixed inset-0 z-[60] flex flex-col justify-end bg-black/80 animate-in fade-in">"""

content = content.replace(old_single_return, new_single_return)

# Close createPortal for Single Drawer
old_single_end = """        </div>
      </div>
    );
  };"""
new_single_end = """        </div>
      </div>,
      document.body
    );
  };"""

content = content.replace(old_single_end, new_single_end)

with open("frontend/src/components/GuInventory.tsx", "w", encoding="utf-8") as f:
    f.write(content)
