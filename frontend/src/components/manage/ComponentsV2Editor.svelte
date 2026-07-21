<!--
  Components V2 editor for the multi-panel ("select category to open a ticket") message.

  Designs a Discord "Components V2" message as an ordered list of blocks — text, separators,
  media galleries, sections (text + an optional image / button accessory), button rows, and
  accent-coloured containers that group leaf blocks. Buttons come in two kinds: a "link" button
  (opens a URL) and a "ticket" button (opens one of the multi-panel's sub-panels).

  When the panel is in button mode you can place ticket buttons anywhere (as section accessories
  or in button rows) to control the layout; the bot then does NOT auto-append the default button
  row. In dropdown mode the select menu is always appended at the end (Discord can't put a select
  inside a section). Accent colour is optional.

  The model written to `data.components` matches the backend's semantic schema exactly.
-->
<div class="cv2">
    <details class="cv2-io">
        <summary><i class="fas fa-code"></i> Import / Export JSON</summary>
        <div class="cv2-io-body">
            <textarea class="cv2-io-text" bind:value={ioText} spellcheck="false"
                placeholder={'Paste a Components V2 JSON layout here and click Import, or click "Export current" to dump the current layout.'}></textarea>
            <div class="cv2-io-actions">
                <Button icon="fas fa-file-export" type="button" on:click={doExport}>Export current</Button>
                <Button icon="fas fa-file-import" type="button" on:click={doImport}>Import</Button>
                {#if ioMsg}<span class="cv2-io-msg" class:err={ioErr}>{ioMsg}</span>{/if}
            </div>
        </div>
    </details>

    <div class="cv2-cols">
        <!-- ============ Editor ============ -->
        <div class="cv2-col">
            {#if blocks.length === 0}
                <p class="cv2-muted">No components yet — add one below to start building your message.</p>
            {/if}

            {#each blocks as block, i (block._id)}
                <div class="cv2-block">
                    <div class="cv2-head">
                        <span class="cv2-type"><i class="fas {icon(block.type)}"></i> {name(block.type)}</span>
                        <div class="cv2-actions">
                            <button type="button" title="Move up" disabled={i === 0} on:click={() => move(blocks, i, -1)}><i class="fas fa-arrow-up"></i></button>
                            <button type="button" title="Move down" disabled={i === blocks.length - 1} on:click={() => move(blocks, i, 1)}><i class="fas fa-arrow-down"></i></button>
                            <button type="button" class="danger" title="Remove" on:click={() => removeAt(blocks, i)}><i class="fas fa-trash"></i></button>
                        </div>
                    </div>

                    {#if block.type === 'container'}
                        <div class="cv2-accent">
                            <Checkbox label="Accent colour" bind:value={block._accentOn} on:change={commit}/>
                            {#if block._accentOn}
                                <input type="color" bind:value={block._accentHex} on:input={commit}/>
                            {/if}
                        </div>

                        <div class="cv2-children">
                            {#each block.children as child, ci (child._id)}
                                <div class="cv2-child">
                                    <div class="cv2-head">
                                        <span class="cv2-type"><i class="fas {icon(child.type)}"></i> {name(child.type)}</span>
                                        <div class="cv2-actions">
                                            <button type="button" title="Move up" disabled={ci === 0} on:click={() => move(block.children, ci, -1)}><i class="fas fa-arrow-up"></i></button>
                                            <button type="button" title="Move down" disabled={ci === block.children.length - 1} on:click={() => move(block.children, ci, 1)}><i class="fas fa-arrow-down"></i></button>
                                            <button type="button" class="danger" title="Remove" on:click={() => removeAt(block.children, ci)}><i class="fas fa-trash"></i></button>
                                        </div>
                                    </div>
                                    <BlockFields block={child} {subPanels} on:change={commit}/>
                                </div>
                            {/each}
                            <div class="cv2-addrow">
                                <select bind:value={block._add}>
                                    {#each leafKinds as k}<option value={k.v}>{k.t}</option>{/each}
                                </select>
                                <Button icon="fas fa-plus" type="button" on:click={() => addChild(block)}>Add</Button>
                            </div>
                        </div>
                    {:else}
                        <BlockFields {block} {subPanels} on:change={commit}/>
                    {/if}
                </div>
            {/each}

            <div class="cv2-add">
                <Dropdown col1 label="Add Component" bind:value={addKind}>
                    {#each topKinds as k}<option value={k.v}>{k.t}</option>{/each}
                </Dropdown>
                <Button icon="fas fa-plus" type="button" on:click={() => add(addKind)}>Add</Button>
            </div>
        </div>

        <!-- ============ Preview ============ -->
        <div class="cv2-col">
            <label class="form-label">Live Preview</label>
            <div class="cv2-preview">
                {#each blocks as block (block._id)}
                    <Preview {block} {subPanels}/>
                {/each}

                {#if selectMenu}
                    <div class="cv2-mock-select"><span>{placeholder || 'Select a topic...'}</span><i class="fas fa-chevron-down"></i></div>
                {:else if !hasTicketButton}
                    <div class="cv2-mock-buttons">
                        {#each (subPanels.length ? subPanels : [{label: 'Open a ticket'}]) as p}
                            <span class="cv2-mock-btn">{p.label}</span>
                        {/each}
                    </div>
                {/if}
            </div>
            <p class="cv2-muted">
                {#if selectMenu}
                    The category dropdown is appended automatically.
                {:else if hasTicketButton}
                    You've placed ticket buttons, so no extra button row is added.
                {:else}
                    The category buttons are appended automatically — add a ticket button anywhere to place them yourself.
                {/if}
            </p>
        </div>
    </div>
</div>

<script context="module">
    let uid = 1;
    export function newId() { return uid++; }
</script>

<script>
    import { onMount } from "svelte";
    import Checkbox from "../form/Checkbox.svelte";
    import Dropdown from "../form/Dropdown.svelte";
    import Button from "../Button.svelte";
    import BlockFields from "./cv2/BlockFields.svelte";
    import Preview from "./cv2/Cv2Preview.svelte";

    export let data;
    export let enabled = false;
    export let selectMenu = false;
    export let placeholder = "";
    export let subPanels = []; // [{ id, label }]

    const DEFAULT_ACCENT = "#2ecc71";

    const topKinds = [
        { v: "text", t: "Text" },
        { v: "section", t: "Section (text + accessory)" },
        { v: "buttons", t: "Button Row" },
        { v: "gallery", t: "Media Gallery" },
        { v: "separator", t: "Separator" },
        { v: "container", t: "Container (accent card)" },
    ];
    const leafKinds = topKinds.filter((k) => k.v !== "container");

    let blocks = [];
    let addKind = "text";
    let initialized = false;

    function name(t) {
        return { text: "Text", separator: "Separator", section: "Section", buttons: "Button Row", gallery: "Media Gallery", container: "Container" }[t] || t;
    }
    function icon(t) {
        return { text: "fa-align-left", separator: "fa-minus", section: "fa-image", buttons: "fa-hand-pointer", gallery: "fa-images", container: "fa-square-full" }[t] || "fa-cube";
    }

    function hexToInt(hex) { return hex ? parseInt(hex.replace("#", ""), 16) : null; }
    function intToHex(n) { return "#" + ((n ?? 0) & 0xffffff).toString(16).padStart(6, "0"); }

    // ---- new blocks ----
    function makeLeaf(type) {
        switch (type) {
            case "text": return { _id: newId(), type: "text", content: "" };
            case "separator": return { _id: newId(), type: "separator", divider: true, spacing: "small" };
            case "gallery": return { _id: newId(), type: "gallery", items: [{ url: "" }] };
            case "section": return { _id: newId(), type: "section", content: "", _accessory: "none", _image: "", _button: makeButton() };
            case "buttons": return { _id: newId(), type: "buttons", buttons: [makeButton()] };
        }
    }
    function makeButton() {
        return { _id: newId(), kind: "link", label: "", emoji: "", url: "", panelId: subPanels[0]?.id ?? null, style: "primary" };
    }
    function add(type) {
        const block = type === "container"
            ? { _id: newId(), type: "container", _accentOn: false, _accentHex: DEFAULT_ACCENT, _add: "text", children: [] }
            : makeLeaf(type);
        blocks = [...blocks, block];
        commit();
    }
    function addChild(container) {
        container.children = [...container.children, makeLeaf(container._add)];
        commit();
    }

    // ---- list ops (mutate array in place, then commit reassigns) ----
    function removeAt(arr, i) { arr.splice(i, 1); commit(); }
    function move(arr, i, dir) {
        const j = i + dir;
        if (j < 0 || j >= arr.length) return;
        [arr[i], arr[j]] = [arr[j], arr[i]];
        commit();
    }

    // ---- serialization (editor model -> wire model) ----
    function buttonToWire(b) {
        if (b.kind === "ticket") {
            const out = { kind: "ticket", panelId: b.panelId ?? null };
            // panelLabel is a portability hint: the backend ignores it, but on import it lets a
            // ticket button re-bind to the matching sub-panel by name.
            const p = subPanels.find((x) => x.id === b.panelId);
            if (p) out.panelLabel = p.label;
            if (b.label) out.label = b.label;
            if (b.emoji) out.emoji = b.emoji;
            if (b.style) out.style = b.style;
            return out;
        }
        const out = { kind: "link", label: b.label || "", url: b.url || "" };
        if (b.emoji) out.emoji = b.emoji;
        return out;
    }
    function leafToWire(b) {
        switch (b.type) {
            case "text": return { type: "text", content: b.content || "" };
            case "separator": return { type: "separator", divider: b.divider !== false, spacing: b.spacing === "large" ? "large" : "small" };
            case "gallery": return { type: "gallery", items: (b.items || []).filter((it) => it.url).map((it) => ({ url: it.url })) };
            case "buttons": return { type: "buttons", buttons: (b.buttons || []).map(buttonToWire) };
            case "section": {
                const out = { type: "section", content: b.content || "" };
                if (b._accessory === "image") out.accessory = { kind: "image", url: b._image || "" };
                else if (b._accessory === "button") out.accessory = { kind: "button", button: buttonToWire(b._button) };
                return out;
            }
        }
    }
    function toWire(b) {
        if (b.type === "container") {
            const out = { type: "container", children: (b.children || []).map(leafToWire) };
            if (b._accentOn) out.accentColor = hexToInt(b._accentHex);
            return out;
        }
        return leafToWire(b);
    }

    // ---- deserialization (wire model -> editor model) ----
    function resolvePanelId(b) {
        const validIds = new Set(subPanels.map((p) => p.id));
        let id = b?.panelId ?? null;
        if ((id == null || !validIds.has(id)) && b?.panelLabel) {
            const m = subPanels.find((p) => (p.label || "").toLowerCase() === String(b.panelLabel).toLowerCase());
            if (m) id = m.id;
        }
        if (id == null || !validIds.has(id)) id = subPanels[0]?.id ?? null;
        return id;
    }

    function buttonFromWire(b) {
        return {
            _id: newId(),
            kind: b?.kind === "ticket" ? "ticket" : "link",
            label: b?.label || "",
            emoji: b?.emoji || "",
            url: b?.url || "",
            panelId: resolvePanelId(b),
            style: b?.style || "primary",
        };
    }
    function leafFromWire(b) {
        switch (b.type) {
            case "text": return { _id: newId(), type: "text", content: b.content || "" };
            case "separator": return { _id: newId(), type: "separator", divider: b.divider !== false, spacing: b.spacing === "large" ? "large" : "small" };
            case "gallery": return { _id: newId(), type: "gallery", items: (b.items?.length ? b.items.map((it) => ({ url: it.url || "" })) : [{ url: "" }]) };
            case "buttons": return { _id: newId(), type: "buttons", buttons: (b.buttons?.length ? b.buttons.map(buttonFromWire) : [makeButton()]) };
            case "section": {
                const s = { _id: newId(), type: "section", content: b.content || "", _accessory: "none", _image: "", _button: makeButton() };
                if (b.accessory?.kind === "image") { s._accessory = "image"; s._image = b.accessory.url || ""; }
                else if (b.accessory?.kind === "button") { s._accessory = "button"; s._button = buttonFromWire(b.accessory.button); }
                return s;
            }
            default: return null;
        }
    }
    function fromWire(b) {
        if (b?.type === "container") {
            return {
                _id: newId(),
                type: "container",
                _accentOn: typeof b.accentColor === "number",
                _accentHex: typeof b.accentColor === "number" ? intToHex(b.accentColor) : DEFAULT_ACCENT,
                _add: "text",
                children: (b.children || []).map(leafFromWire).filter(Boolean),
            };
        }
        return leafFromWire(b);
    }

    function commit() {
        blocks = blocks; // trigger reactivity for the preview
        if (enabled) data.components = blocks.map(toWire);
    }

    // ---- import / export ----
    let ioText = "";
    let ioMsg = "";
    let ioErr = false;

    function doExport() {
        ioText = JSON.stringify(blocks.map(toWire), null, 2);
        ioErr = false;
        ioMsg = "Current layout exported below — copy it to save or share.";
    }

    function doImport() {
        ioErr = false;
        ioMsg = "";
        let parsed;
        try {
            parsed = JSON.parse(ioText);
        } catch (e) {
            ioErr = true;
            ioMsg = "Invalid JSON: " + e.message;
            return;
        }
        if (!Array.isArray(parsed)) {
            if (parsed && typeof parsed === "object") parsed = [parsed];
            else {
                ioErr = true;
                ioMsg = "The JSON must be an array of components.";
                return;
            }
        }
        const next = parsed.map(fromWire).filter(Boolean);
        if (next.length === 0) {
            ioErr = true;
            ioMsg = "No valid components were found in that JSON.";
            return;
        }
        blocks = next;
        commit();
        ioMsg = `Imported ${next.length} top-level component(s).`;
    }

    // Does the layout place any ticket button? (drives the mock preview + hint)
    function scanTicket(list) {
        for (const b of list || []) {
            if (b.type === "buttons" && (b.buttons || []).some((x) => x.kind === "ticket")) return true;
            if (b.type === "section" && b._accessory === "button" && b._button?.kind === "ticket") return true;
            if (b.type === "container" && scanTicket(b.children)) return true;
        }
        return false;
    }
    $: hasTicketButton = scanTicket(blocks);

    // keep data.components in sync when the mode / contents change
    $: if (initialized) data.components = enabled ? blocks.map(toWire) : null;

    onMount(() => {
        const comps = data && data.components;
        if (Array.isArray(comps) && comps.length > 0) {
            blocks = comps.map(fromWire).filter(Boolean);
        }
        initialized = true;
    });
</script>

<style>
    .cv2 { width: 100%; }

    .cv2-io { margin-bottom: 12px; border: 1px solid var(--background-secondary, #3a3d44); border-radius: 6px; }
    .cv2-io > summary { cursor: pointer; padding: 8px 12px; font-size: 0.85rem; font-weight: 600; user-select: none; }
    .cv2-io-body { padding: 0 12px 12px; display: flex; flex-direction: column; gap: 8px; }
    .cv2-io-text {
        width: 100%; min-height: 120px; resize: vertical; font-family: monospace; font-size: 0.8rem;
        background: #1e1f22; color: #dbdee1; border: 1px solid var(--background-secondary, #3a3d44);
        border-radius: 6px; padding: 8px;
    }
    .cv2-io-actions { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
    .cv2-io-msg { font-size: 0.82rem; color: #57f287; }
    .cv2-io-msg.err { color: #f04747; }

    .cv2-cols { display: flex; gap: 20px; width: 100%; }
    .cv2-col { display: flex; flex-direction: column; gap: 10px; width: 50%; min-width: 0; }
    .cv2-muted { color: var(--text-muted, #99aab5); font-size: 0.85rem; font-style: italic; margin: 4px 0; }

    .cv2-block, .cv2-child {
        border: 1px solid var(--background-secondary, #3a3d44);
        border-radius: 6px; padding: 10px 12px; display: flex; flex-direction: column; gap: 8px;
    }
    .cv2-child { background: rgba(0,0,0,0.06); }
    .cv2-head { display: flex; justify-content: space-between; align-items: center; }
    .cv2-type { font-weight: 600; font-size: 0.9rem; }
    .cv2-actions { display: flex; gap: 6px; }
    .cv2-actions button {
        background: var(--background-secondary, #3a3d44); border: none; color: inherit;
        border-radius: 4px; padding: 5px 8px; cursor: pointer; font-size: 0.8rem;
    }
    .cv2-actions button:disabled { opacity: 0.4; cursor: not-allowed; }
    .cv2-actions button.danger { color: #f04747; }

    .cv2-accent { display: flex; align-items: center; gap: 10px; }
    .cv2-accent input[type=color] { width: 42px; height: 28px; border: none; background: none; padding: 0; cursor: pointer; }

    .cv2-children { display: flex; flex-direction: column; gap: 8px; padding-left: 8px; border-left: 2px solid var(--background-secondary, #3a3d44); }
    .cv2-addrow, .cv2-add { display: flex; align-items: flex-end; gap: 10px; }
    .cv2-addrow select { background: var(--background-secondary, #3a3d44); color: inherit; border: none; border-radius: 4px; padding: 8px; }
    .cv2-add :global(.col-1) { flex: 1; max-width: 280px; }

    .cv2-preview { background: #313338; border-radius: 8px; padding: 16px; min-height: 120px; display: flex; flex-direction: column; gap: 8px; }
    .cv2-mock-select {
        display: flex; justify-content: space-between; align-items: center; background: #1e1f22;
        border: 1px solid #232428; border-radius: 4px; padding: 8px 12px; color: #949ba4; font-size: 0.88rem;
    }
    .cv2-mock-buttons { display: flex; flex-wrap: wrap; gap: 6px; }
    .cv2-mock-btn { background: #5865f2; color: #fff; border-radius: 4px; padding: 6px 12px; font-size: 0.85rem; }

    @media only screen and (max-width: 950px) {
        .cv2-cols { flex-direction: column; }
        .cv2-col { width: 100%; }
    }
</style>
