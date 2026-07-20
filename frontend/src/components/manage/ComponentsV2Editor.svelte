<!--
  Components V2 editor for the multi-panel ("select category to open a ticket") message.

  Lets the user design a rich Discord "Components V2" message (an accent-coloured container
  holding text displays, sections, media galleries and separators) with a live preview. The
  category picker (dropdown / buttons) is added by the bot automatically, so it is only shown
  as a mock at the bottom of the preview.

  The editor reads and writes `components` — the JSON array persisted on the multi-panel. When
  enabled it produces `[{ type: 17 (container), accent_color, components: [...] }]`; when
  disabled it is set to null and the classic embed is used instead.
-->
<div class="cv2-editor">
    <div class="cv2-columns">
        <!-- Editor column -->
        <div class="cv2-col">
            <div class="cv2-accent-row">
                <Colour col1 label="Accent Colour" bind:value={accentHex} on:input={commit} on:change={commit}/>
            </div>

            {#if blocks.length === 0}
                <p class="cv2-empty">No components yet. Add one below to start building your message.</p>
            {/if}

            {#each blocks as block, i (block.id)}
                <div class="cv2-block">
                    <div class="cv2-block-header">
                        <span class="cv2-block-type"><i class="fas {blockIcon(block.kind)}"></i> {blockName(block.kind)}</span>
                        <div class="cv2-block-actions">
                            <button type="button" title="Move up" disabled={i === 0} on:click={() => move(i, -1)}>
                                <i class="fas fa-arrow-up"></i>
                            </button>
                            <button type="button" title="Move down" disabled={i === blocks.length - 1} on:click={() => move(i, 1)}>
                                <i class="fas fa-arrow-down"></i>
                            </button>
                            <button type="button" class="danger" title="Remove" on:click={() => remove(i)}>
                                <i class="fas fa-trash"></i>
                            </button>
                        </div>
                    </div>

                    {#if block.kind === 'text'}
                        <Textarea col1 label="Text (Markdown supported)" placeholder="Write some text…"
                                  bind:value={block.content} minHeight="70px"/>
                    {:else if block.kind === 'separator'}
                        <p class="cv2-hint">A horizontal divider line.</p>
                    {:else if block.kind === 'section'}
                        <Textarea col1 label="Text (Markdown supported)" placeholder="Write some text…"
                                  bind:value={block.content} minHeight="70px"/>
                        <Input col1 label="Thumbnail Image URL" placeholder="https://…" bind:value={block.imageUrl}/>
                    {:else if block.kind === 'media'}
                        {#each block.items as item, j}
                            <div class="cv2-media-row">
                                <Input col1 label={j === 0 ? 'Image URLs' : undefined} placeholder="https://…" bind:value={item.url}/>
                                <button type="button" class="danger cv2-media-remove" title="Remove image"
                                        disabled={block.items.length === 1} on:click={() => removeMediaItem(block, j)}>
                                    <i class="fas fa-times"></i>
                                </button>
                            </div>
                        {/each}
                        <button type="button" class="cv2-add-image" disabled={block.items.length >= 10}
                                on:click={() => addMediaItem(block)}>
                            <i class="fas fa-plus"></i> Add image
                        </button>
                    {/if}
                </div>
            {/each}

            <div class="cv2-add">
                <Dropdown col1 label="Add Component" bind:value={addKind}>
                    <option value="text">Text</option>
                    <option value="section">Section (text + image)</option>
                    <option value="media">Media Gallery</option>
                    <option value="separator">Separator</option>
                </Dropdown>
                <Button icon="fas fa-plus" type="button" on:click={() => add(addKind)}>Add</Button>
            </div>
        </div>

        <!-- Preview column -->
        <div class="cv2-col">
            <label class="form-label">Live Preview</label>
            <div class="cv2-preview">
                <div class="cv2-container" style="border-left-color: {accentHex};">
                    {#each blocks as block (block.id)}
                        {#if block.kind === 'text'}
                            <div class="cv2-text">{@html renderMarkdown(block.content)}</div>
                        {:else if block.kind === 'separator'}
                            <div class="cv2-separator"></div>
                        {:else if block.kind === 'section'}
                            <div class="cv2-section">
                                <div class="cv2-text">{@html renderMarkdown(block.content)}</div>
                                {#if block.imageUrl}
                                    <img class="cv2-thumb" src={block.imageUrl} alt="thumbnail"/>
                                {/if}
                            </div>
                        {:else if block.kind === 'media'}
                            <div class="cv2-gallery">
                                {#each block.items.filter(it => it.url) as item}
                                    <img src={item.url} alt="media"/>
                                {/each}
                            </div>
                        {/if}
                    {/each}

                    <!-- Mock category picker (added by the bot) -->
                    {#if selectMenu}
                        <div class="cv2-mock-select">
                            <span>{placeholder || 'Select a topic...'}</span>
                            <i class="fas fa-chevron-down"></i>
                        </div>
                    {:else}
                        <div class="cv2-mock-buttons">
                            {#each (panelLabels && panelLabels.length ? panelLabels : ['Open a ticket']) as label}
                                <span class="cv2-mock-button">{label}</span>
                            {/each}
                        </div>
                    {/if}
                </div>
            </div>
            <p class="cv2-hint">The category {selectMenu ? 'dropdown' : 'buttons'} shown above is added automatically.</p>
        </div>
    </div>
</div>

<script>
    import { onMount } from "svelte";
    import Colour from "../form/Colour.svelte";
    import Input from "../form/Input.svelte";
    import Textarea from "../form/Textarea.svelte";
    import Dropdown from "../form/Dropdown.svelte";
    import Button from "../Button.svelte";

    // The multi-panel form data. We read/write `data.components`.
    export let data;
    export let enabled = false;

    // Preview context (mirrors how the bot appends the picker).
    export let selectMenu = false;
    export let placeholder = "";
    export let panelLabels = [];

    const DEFAULT_ACCENT = "#2ecc71";

    let accentHex = DEFAULT_ACCENT;
    let blocks = [];
    let addKind = "text";
    let initialized = false;
    let nextId = 1;

    function newId() {
        return nextId++;
    }

    function intToHex(n) {
        if (n === null || n === undefined) return DEFAULT_ACCENT;
        return "#" + (n & 0xffffff).toString(16).padStart(6, "0");
    }

    function hexToInt(hex) {
        if (!hex) return null;
        return parseInt(hex.replace("#", ""), 16);
    }

    // Build the wire (Discord) representation of a single editor block.
    function toWire(block) {
        switch (block.kind) {
            case "text":
                return { type: 10, content: block.content || "" };
            case "separator":
                return { type: 14 };
            case "section":
                return {
                    type: 9,
                    components: [{ type: 10, content: block.content || "" }],
                    accessory: { type: 11, media: { url: block.imageUrl || "" } },
                };
            case "media":
                return {
                    type: 12,
                    items: (block.items || [])
                        .filter((it) => it.url)
                        .map((it) => ({ media: { url: it.url } })),
                };
            default:
                return null;
        }
    }

    // Parse a stored container child back into an editor block.
    function fromWire(c) {
        if (!c) return null;
        switch (c.type) {
            case 10:
                return { id: newId(), kind: "text", content: c.content || "" };
            case 14:
                return { id: newId(), kind: "separator" };
            case 9:
                return {
                    id: newId(),
                    kind: "section",
                    content: (c.components && c.components[0] && c.components[0].content) || "",
                    imageUrl: (c.accessory && c.accessory.media && c.accessory.media.url) || "",
                };
            case 12:
                return {
                    id: newId(),
                    kind: "media",
                    items:
                        c.items && c.items.length
                            ? c.items.map((it) => ({ url: (it.media && it.media.url) || "" }))
                            : [{ url: "" }],
                };
            default:
                return null;
        }
    }

    function buildComponents() {
        return [
            {
                type: 17,
                accent_color: hexToInt(accentHex),
                components: blocks.map(toWire).filter(Boolean),
            },
        ];
    }

    // Re-assign blocks (to trigger reactivity/preview) and write back to the form data.
    function commit() {
        blocks = blocks;
        if (enabled) {
            data.components = buildComponents();
        }
    }

    function add(kind) {
        let block;
        switch (kind) {
            case "text":
                block = { id: newId(), kind: "text", content: "" };
                break;
            case "separator":
                block = { id: newId(), kind: "separator" };
                break;
            case "section":
                block = { id: newId(), kind: "section", content: "", imageUrl: "" };
                break;
            case "media":
                block = { id: newId(), kind: "media", items: [{ url: "" }] };
                break;
            default:
                return;
        }
        blocks = [...blocks, block];
        commit();
    }

    function remove(i) {
        blocks = blocks.filter((_, idx) => idx !== i);
        commit();
    }

    function move(i, dir) {
        const j = i + dir;
        if (j < 0 || j >= blocks.length) return;
        const copy = blocks.slice();
        [copy[i], copy[j]] = [copy[j], copy[i]];
        blocks = copy;
        commit();
    }

    function addMediaItem(block) {
        if (block.items.length >= 10) return;
        block.items = [...block.items, { url: "" }];
        commit();
    }

    function removeMediaItem(block, j) {
        if (block.items.length <= 1) return;
        block.items = block.items.filter((_, idx) => idx !== j);
        commit();
    }

    function blockName(kind) {
        return { text: "Text", separator: "Separator", section: "Section", media: "Media Gallery" }[kind] || kind;
    }

    function blockIcon(kind) {
        return { text: "fa-align-left", separator: "fa-minus", section: "fa-image", media: "fa-images" }[kind] || "fa-cube";
    }

    // Minimal Discord-flavoured Markdown rendering for the preview. Input is HTML-escaped
    // first so user content can't inject markup.
    function renderMarkdown(text) {
        if (!text) return "";
        const escaped = text
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;");
        return escaped
            .replace(/^### (.*)$/gm, "<h3>$1</h3>")
            .replace(/^## (.*)$/gm, "<h2>$1</h2>")
            .replace(/^# (.*)$/gm, "<h1>$1</h1>")
            .replace(/^-# (.*)$/gm, '<span class="cv2-subtext">$1</span>')
            .replace(/\*\*(.*?)\*\*/g, "<strong>$1</strong>")
            .replace(/\*(.*?)\*/g, "<em>$1</em>")
            .replace(/__(.*?)__/g, "<u>$1</u>")
            .replace(/~~(.*?)~~/g, "<s>$1</s>")
            .replace(/`(.*?)`/g, "<code>$1</code>")
            .replace(/\n/g, "<br>");
    }

    // Whenever the mode toggles, keep data.components in sync.
    $: if (initialized) {
        if (enabled) {
            data.components = buildComponents();
        } else {
            data.components = null;
        }
    }

    onMount(() => {
        const comps = data && data.components;
        if (Array.isArray(comps) && comps.length > 0) {
            const container = comps.find((c) => c && c.type === 17) || comps[0];
            if (container && container.type === 17) {
                accentHex = intToHex(container.accent_color);
                blocks = (container.components || []).map(fromWire).filter(Boolean);
            }
        }
        initialized = true;
    });
</script>

<style>
    .cv2-editor {
        width: 100%;
    }

    .cv2-columns {
        display: flex;
        flex-direction: row;
        gap: 20px;
        width: 100%;
    }

    .cv2-col {
        display: flex;
        flex-direction: column;
        gap: 10px;
        width: 50%;
        min-width: 0;
    }

    .cv2-accent-row {
        max-width: 200px;
    }

    .cv2-empty,
    .cv2-hint {
        color: var(--text-muted, #99aab5);
        font-size: 0.85rem;
        font-style: italic;
        margin: 4px 0;
    }

    .cv2-block {
        border: 1px solid var(--background-secondary, #3a3d44);
        border-radius: 6px;
        padding: 10px 12px;
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .cv2-block-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .cv2-block-type {
        font-weight: 600;
        font-size: 0.9rem;
    }

    .cv2-block-actions {
        display: flex;
        gap: 6px;
    }

    .cv2-block-actions button,
    .cv2-add-image,
    .cv2-media-remove {
        background: var(--background-secondary, #3a3d44);
        border: none;
        color: inherit;
        border-radius: 4px;
        padding: 5px 8px;
        cursor: pointer;
        font-size: 0.8rem;
    }

    .cv2-block-actions button:disabled,
    .cv2-add-image:disabled,
    .cv2-media-remove:disabled {
        opacity: 0.4;
        cursor: not-allowed;
    }

    .cv2-block-actions button.danger,
    .cv2-media-remove.danger {
        color: #f04747;
    }

    .cv2-media-row {
        display: flex;
        align-items: flex-end;
        gap: 6px;
    }

    .cv2-media-row :global(.col-1) {
        flex: 1;
    }

    .cv2-add-image {
        align-self: flex-start;
    }

    .cv2-add {
        display: flex;
        align-items: flex-end;
        gap: 10px;
        margin-top: 6px;
    }

    .cv2-add :global(.col-1) {
        flex: 1;
        max-width: 260px;
    }

    /* Preview */
    .cv2-preview {
        background-color: #313338;
        border-radius: 8px;
        padding: 16px;
        min-height: 120px;
    }

    .cv2-container {
        background-color: #2b2d31;
        border-left: 4px solid #2ecc71;
        border-radius: 4px;
        padding: 12px 14px;
        display: flex;
        flex-direction: column;
        gap: 8px;
        max-width: 460px;
        color: #dbdee1;
    }

    .cv2-text {
        font-size: 0.9rem;
        line-height: 1.4;
        word-wrap: break-word;
    }

    .cv2-text :global(h1) { font-size: 1.3rem; margin: 2px 0; }
    .cv2-text :global(h2) { font-size: 1.15rem; margin: 2px 0; }
    .cv2-text :global(h3) { font-size: 1rem; margin: 2px 0; }
    .cv2-text :global(code) {
        background: rgba(0, 0, 0, 0.3);
        padding: 1px 4px;
        border-radius: 3px;
        font-family: monospace;
    }
    .cv2-text :global(.cv2-subtext) {
        font-size: 0.78rem;
        color: #949ba4;
    }

    .cv2-separator {
        height: 1px;
        background-color: rgba(255, 255, 255, 0.1);
        margin: 2px 0;
    }

    .cv2-section {
        display: flex;
        justify-content: space-between;
        gap: 12px;
        align-items: flex-start;
    }

    .cv2-thumb {
        width: 72px;
        height: 72px;
        object-fit: cover;
        border-radius: 6px;
        flex-shrink: 0;
    }

    .cv2-gallery {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(90px, 1fr));
        gap: 6px;
    }

    .cv2-gallery img {
        width: 100%;
        height: 90px;
        object-fit: cover;
        border-radius: 6px;
    }

    .cv2-mock-select {
        display: flex;
        justify-content: space-between;
        align-items: center;
        background-color: #1e1f22;
        border: 1px solid #232428;
        border-radius: 4px;
        padding: 8px 12px;
        color: #949ba4;
        font-size: 0.88rem;
        margin-top: 4px;
    }

    .cv2-mock-buttons {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
        margin-top: 4px;
    }

    .cv2-mock-button {
        background-color: #5865f2;
        color: #fff;
        border-radius: 4px;
        padding: 6px 12px;
        font-size: 0.85rem;
    }

    @media only screen and (max-width: 950px) {
        .cv2-columns {
            flex-direction: column;
        }

        .cv2-col {
            width: 100%;
        }
    }
</style>
