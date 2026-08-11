import re

with open("frontend/src/components/GuInventory.tsx", "r", encoding="utf-8") as f:
    content = f.read()

with open("gu_drawers.tsx", "r", encoding="utf-8") as f:
    drawers_content = f.read()

with open("gu_modals.tsx", "r", encoding="utf-8") as f:
    modals_content = f.read()

# Replace renderGroup
# Find start
start_str = "              const renderGroup = ({"
end_str = "              return (\n                <div className=\"space-y-6\">"
# Use string slice to replace
start_idx = content.find(start_str)
end_idx = content.find(end_str)

if start_idx != -1 and end_idx != -1:
    content = content[:start_idx] + drawers_content + "\n" + content[end_idx:]
else:
    print("Could not find renderGroup bounds")

# Insert modal render functions before the main return
main_return_start = content.find("  return (\n    <div className=\"glass-panel-gold")
if main_return_start != -1:
    content = content[:main_return_start] + modals_content + "\n" + content[main_return_start:]
else:
    print("Could not find main return")

# Insert `{renderGroupDrawer()} {renderSingleInstanceDrawer()}` at the very end of the main div
final_div_end = content.rfind("    </div>\n  );\n};")
if final_div_end != -1:
    insert_code = """
      {renderGroupDrawer()}
      {renderSingleInstanceDrawer()}
"""
    content = content[:final_div_end] + insert_code + content[final_div_end:]
else:
    print("Could not find final div end")

with open("frontend/src/components/GuInventory.tsx", "w", encoding="utf-8") as f:
    f.write(content)

