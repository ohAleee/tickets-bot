<!-- Edit fields for a single leaf block (text / separator / gallery / section / buttons). -->
{#if block.type === 'text'}
    <Textarea col1 label="Text (Markdown supported)" placeholder="Write some text…" minHeight="70px" bind:value={block.content} on:input={changed}/>

{:else if block.type === 'separator'}
    <div class="line">
        <Checkbox label="Show divider line" bind:value={block.divider} on:change={changed}/>
        <label class="mini">Spacing</label>
        <select bind:value={block.spacing} on:change={changed}>
            <option value="small">Small</option>
            <option value="large">Large</option>
        </select>
    </div>

{:else if block.type === 'gallery'}
    {#each block.items as item, i}
        <div class="line">
            <Input col1 label={i === 0 ? 'Image URLs' : undefined} placeholder="https://…" bind:value={item.url} on:input={changed}/>
            <button type="button" class="rm" title="Remove" disabled={block.items.length === 1} on:click={() => removeItem(i)}><i class="fas fa-times"></i></button>
        </div>
    {/each}
    <button type="button" class="add" disabled={block.items.length >= 10} on:click={addItem}><i class="fas fa-plus"></i> Add image</button>

{:else if block.type === 'section'}
    <Textarea col1 label="Text (Markdown supported)" placeholder="Write some text…" minHeight="70px" bind:value={block.content} on:input={changed}/>
    <div class="line">
        <label class="mini">Accessory</label>
        <select bind:value={block._accessory} on:change={changed}>
            <option value="none">None</option>
            <option value="image">Image (thumbnail)</option>
            <option value="button">Button</option>
        </select>
    </div>
    {#if block._accessory === 'image'}
        <Input col1 label="Image URL" placeholder="https://…" bind:value={block._image} on:input={changed}/>
    {:else if block._accessory === 'button'}
        <ButtonEditor button={block._button} {subPanels} on:change={changed}/>
    {/if}

{:else if block.type === 'buttons'}
    {#each block.buttons as btn, i (btn._id)}
        <div class="btnrow">
            <ButtonEditor button={btn} {subPanels} on:change={changed}/>
            <button type="button" class="rm" title="Remove button" disabled={block.buttons.length === 1} on:click={() => removeButton(i)}><i class="fas fa-times"></i></button>
        </div>
    {/each}
    <button type="button" class="add" disabled={block.buttons.length >= 5} on:click={addButton}><i class="fas fa-plus"></i> Add button</button>
{/if}

<script>
    import { createEventDispatcher } from "svelte";
    import Input from "../../form/Input.svelte";
    import Textarea from "../../form/Textarea.svelte";
    import Checkbox from "../../form/Checkbox.svelte";
    import ButtonEditor from "./ButtonEditor.svelte";

    export let block;
    export let subPanels = [];

    const dispatch = createEventDispatcher();
    function changed() { dispatch("change"); }

    let bid = 1;
    function newButton() {
        return { _id: `b${Date.now()}-${bid++}`, kind: "link", label: "", emoji: "", url: "", panelId: subPanels[0]?.id ?? null };
    }

    function addItem() { block.items = [...block.items, { url: "" }]; changed(); }
    function removeItem(i) { block.items.splice(i, 1); block.items = block.items; changed(); }
    function addButton() { block.buttons = [...block.buttons, newButton()]; changed(); }
    function removeButton(i) { block.buttons.splice(i, 1); block.buttons = block.buttons; changed(); }
</script>

<style>
    .line { display: flex; align-items: flex-end; gap: 8px; }
    .line :global(.col-1) { flex: 1; }
    .mini { font-size: 0.8rem; color: var(--text-muted, #99aab5); margin-bottom: 6px; }
    select { background: var(--background-secondary, #3a3d44); color: inherit; border: none; border-radius: 4px; padding: 8px; }
    .btnrow { display: flex; align-items: flex-start; gap: 6px; }
    .btnrow > :global(.btn-editor) { flex: 1; }
    .rm, .add {
        background: var(--background-secondary, #3a3d44); border: none; color: inherit;
        border-radius: 4px; padding: 6px 10px; cursor: pointer; font-size: 0.8rem;
    }
    .rm { color: #f04747; }
    .rm:disabled, .add:disabled { opacity: 0.4; cursor: not-allowed; }
    .add { align-self: flex-start; }
</style>
