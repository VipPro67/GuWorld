import re
import os
import json
import subprocess
import time
import asyncio

log("=== Starting Master Gu Icon Generation Workflow ===")

manifest_path = "/home/vippro67/GitHub/GuWorld/scripts/gu_manifest.json"
output_dir = "/home/vippro67/GitHub/GuWorld/frontend/public/assets/gus"
status_path = "/home/vippro67/GitHub/GuWorld/scripts/gu_generation_status.json"
os.makedirs(output_dir, exist_ok=True)

with open(manifest_path, "r", encoding="utf-8") as f:
    all_gu = json.load(f)

log(f"Loaded {len(all_gu)} total Gu from manifest.")

def get_dest_path(item):
    gid = item.get("id")
    rank = item.get("rank")
    return os.path.join(output_dir, f"{gid}_{rank}.png")

def is_already_done(item):
    dest = get_dest_path(item)
    return os.path.exists(dest) and os.path.getsize(dest) > 1000

init_status = {}
if os.path.exists(status_path):
    try:
        with open(status_path, "r", encoding="utf-8") as f:
            init_status = json.load(f)
    except Exception:
        pass
init_failed = init_status.get("failed", {})

pending_gu = [item for item in all_gu if not is_already_done(item)]
# Sort pending so items with multiple past failures are moved to the back
pending_gu.sort(key=lambda it: init_failed.get(f"{it.get('id')}_{it.get('rank')}", 0))

already_done_count = len(all_gu) - len(pending_gu)
log(f"Status: {already_done_count}/{len(all_gu)} already done. {len(pending_gu)} pending.")

def enhance_prompt(item):
    name = item.get("name", "")
    gid = item.get("id", "")
    rank = item.get("rank", 1)
    orig_prompt = item.get("prompt", "")
    
    theme_match = re.search(r"Theme:\s*([^.]+)\.", orig_prompt)
    grade_match = re.search(r"Grade:\s*([^.]+)\.", orig_prompt)
    
    theme_desc = theme_match.group(1).strip() if theme_match else "mystical Daoist hues"
    grade_desc = grade_match.group(1).strip() if grade_match else f"Rank {rank} Gu"
    
    return (
        f"Masterpiece game inventory icon of '{name}' ({gid}), {grade_desc} from Gu Master universe. "
        f"Theme: {theme_desc}. "
        "Exquisite mystical creature sculpted from translucent celestial jadeite, amber crystal, and glowing Dao marks. "
        "Gossamer wings with intricate flowing Dao mark ribbons, fine luminous aura particles, "
        "cinematic lighting, detailed subsurface scattering, octane render 8k quality, "
        "isolated on solid pure white background (#ffffff), centered isometric view, flat clean cutout edges, no floor shadow, no scenery."
    )

quota_hit = False

async def generate_single_gu(item):
    global quota_hit
    gid = item.get("id")
    rank = item.get("rank")
    name = item.get("name")
    dest_path = get_dest_path(item)
    
    if is_already_done(item):
        return {"id": gid, "rank": rank, "status": "skipped", "path": dest_path}
        
    prompt = enhance_prompt(item)
    image_name = f"gu_{gid.lower()}_{rank}"
    
    def extract_image_path(text):
        for m in re.finditer(r"(/home/[^\s`\*\)\"]+\.(?:jpg|jpeg|png))", text):
            p = m.group(1).strip()
            if os.path.exists(p) and os.path.getsize(p) > 1000:
                return p
        return None

    try:
        agent_instruction = f"Generate image:\nAspectRatio: 1:1\nImageName: {image_name}\nPrompt: {prompt}\nNote: Generate once and finish immediately. Do not attempt follow-up regenerations."
        gen_agent = agent(
            prompt=agent_instruction,
            schema={"type": "string"},
            type_name="image-generator",
            role=f"{name} (R{rank})"
        )
        
        resp = await gen_agent
        src_img = extract_image_path(resp)
        
        if not src_img and ("429" in resp or "RESOURCE_EXHAUSTED" in resp or "quota" in resp.lower() or "capacity" in resp.lower()):
            log(f"Rate limit notice for {gid}_{rank}: {resp[:120]}. Waiting 60s for RPM window reset...")
            await asyncio.sleep(60)
            gen_agent = agent(
                prompt=agent_instruction,
                schema={"type": "string"},
                type_name="image-generator",
                role=f"{name} (R{rank}) retry"
            )
            resp = await gen_agent
            src_img = extract_image_path(resp)
            
            if not src_img and ("429" in resp or "RESOURCE_EXHAUSTED" in resp or "quota" in resp.lower() or "capacity" in resp.lower()):
                log(f"Rate limit still active for {gid}_{rank}. Waiting another 60s for 2nd retry...")
                await asyncio.sleep(60)
                gen_agent = agent(
                    prompt=agent_instruction,
                    schema={"type": "string"},
                    type_name="image-generator",
                    role=f"{name} (R{rank}) retry 2"
                )
                resp = await gen_agent
                src_img = extract_image_path(resp)
                
                if not src_img and ("429" in resp or "RESOURCE_EXHAUSTED" in resp or "quota" in resp.lower() or "capacity" in resp.lower()):
                    log(f"Quota exhaustion persisted after 2 retries for {gid}_{rank}: {resp[:120]}")
                    quota_hit = True
                    return {"id": gid, "rank": rank, "status": "quota_exhausted", "error": resp}
            
        if not src_img:
            log(f"Failed to find valid image path in response for {gid}_{rank}: {resp[:200]}")
            return {"id": gid, "rank": rank, "status": "error", "error": "No valid image path found"}
        
        # Process background removal and resize
        res = subprocess.run([
            "/home/vippro67/GitHub/GuWorld/scripts/.venv/bin/python",
            "/home/vippro67/GitHub/GuWorld/scripts/process_gu_icon.py",
            src_img,
            dest_path
        ], capture_output=True, text=True)
        
        if res.returncode != 0:
            log(f"Post-processing failed for {gid}_{rank}: {res.stderr}")
            return {"id": gid, "rank": rank, "status": "error", "error": res.stderr}
            
        log(f"Created: {gid}_{rank}.png (size: {os.path.getsize(dest_path)}B)")
        return {"id": gid, "rank": rank, "status": "success", "path": dest_path}
        
    except Exception as e:
        err_str = str(e)
        if "429" in err_str or "RESOURCE_EXHAUSTED" in err_str or "quota" in err_str.lower():
            quota_hit = True
        log(f"Exception during {gid}_{rank}: {err_str}")
        return {"id": gid, "rank": rank, "status": "error", "error": err_str}

batch_size = 15
chunk_size = 1
sleep_delay = 14
if "args" in globals() and isinstance(args, dict):
    batch_size = int(args.get("batch_size", batch_size))
    chunk_size = int(args.get("chunk_size", chunk_size))
    sleep_delay = int(args.get("sleep_delay", sleep_delay))

total_processed = already_done_count
batch_targets = pending_gu[:batch_size]
log(f"Starting batch of up to {len(batch_targets)} items in chunks of {chunk_size} with {sleep_delay}s cooldown...")

all_results = []

for i in range(0, len(batch_targets), chunk_size):
    if quota_hit:
        log("Quota exhausted detected. Halting further chunks in this run.")
        break
        
    chunk = batch_targets[i:i + chunk_size]
    log(f"Processing item {i + 1}/{len(batch_targets)} ({chunk[0].get('id')}_{chunk[0].get('rank')})...")
    results = await parallel(chunk, generate_single_gu)
    all_results.extend(results)
    
    successes = sum(1 for r in results if r.get("status") in ("success", "skipped"))
    total_processed += successes
    log(f"Progress: {total_processed}/{len(all_gu)} total done.")
    
    if quota_hit:
        log("Quota exhausted during chunk. Stopping execution.")
        break

    if i + chunk_size < len(batch_targets):
        await asyncio.sleep(sleep_delay)

# Update status file
existing_status = {}
if os.path.exists(status_path):
    try:
        with open(status_path, "r", encoding="utf-8") as f:
            existing_status = json.load(f)
    except Exception:
        pass

failed_map = existing_status.get("failed", {})

for r in all_results:
    if r.get("status") == "error":
        gid = f"{r.get('id')}_{r.get('rank')}"
        failed_map[gid] = failed_map.get(gid, 0) + 1

with open(status_path, "w", encoding="utf-8") as f:
    json.dump({
        "total": len(all_gu),
        "completed": len([item for item in all_gu if is_already_done(item)]),
        "quota_hit": quota_hit,
        "last_batch_time": time.time(),
        "failed": failed_map
    }, f, indent=2)

log(f"Workflow finished. Total icons in frontend/public/assets/gus: {len(os.listdir(output_dir))}")
