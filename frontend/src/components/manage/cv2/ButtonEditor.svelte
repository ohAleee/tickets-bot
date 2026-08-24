<!-- Edits a single Components V2 button: a "link" button (URL) or a "ticket" button that opens
     one of the multi-panel's sub-panels. -->
<div class="btn-editor">
    <div class="row">
        <label class="mini">Button type</label>
        <select bind:value={button.kind} on:change={changed}>
            <option value="link">Link (URL)</option>
            {#if subPanels.length}<option value="ticket">Open ticket panel</option>{/if}
        </select>
    </div>

    {#if button.kind === 'ticket'}
        <div class="row">
            <label class="mini">Panel</label>
            <select bind:value={button.panelId} on:change={changed}>
                {#each subPanels as p}<option value={p.id}>{p.label}</option>{/each}
            </select>
        </div>
        <div class="row">
            <label class="mini">Colour</label>
            <select bind:value={button.style} on:change={changed}>
                <option value="primary">Blurple (Primary)</option>
                <option value="secondary">Grey (Secondary)</option>
                <option value="success">Green (Success)</option>
                <option value="danger">Red (Danger)</option>
            </select>
        </div>
        <Input col1 label="Label (optional - defaults to the panel's)" placeholder="Open a ticket" bind:value={button.label} on:input={changed}/>
        <Input col1 label="Emoji (optional)" placeholder="📩 or <:name:id>" bind:value={button.emoji} on:input={changed}/>
    {:else}
        <Input col1 label="Label" placeholder="Click me" bind:value={button.label} on:input={changed}/>
        <Input col1 label="URL" placeholder="https://…" bind:value={button.url} on:input={changed}/>
        <Input col1 label="Emoji (optional)" placeholder="🔗 or <:name:id>" bind:value={button.emoji} on:input={changed}/>
    {/if}
</div>

<script>
    import { createEventDispatcher } from "svelte";
    import Input from "../../form/Input.svelte";

    export let button;
    export let subPanels = [];

    const dispatch = createEventDispatcher();
    function changed() { dispatch("change"); }
</script>

<style>
    .btn-editor { display: flex; flex-direction: column; gap: 8px; padding: 8px; border: 1px dashed var(--background-secondary, #3a3d44); border-radius: 6px; }
    .row { display: flex; flex-direction: column; gap: 3px; }
    .mini { font-size: 0.8rem; color: var(--text-muted, #99aab5); }
    select { background: var(--background-secondary, #3a3d44); color: inherit; border: none; border-radius: 4px; padding: 8px; }
</style>
