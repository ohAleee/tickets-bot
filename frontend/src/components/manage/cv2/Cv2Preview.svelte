<!-- Discord-style live preview of one Components V2 block (recurses for containers). -->
{#if block.type === 'text'}
    <div class="txt">{@html md(block.content)}</div>

{:else if block.type === 'separator'}
    <div class="sep" class:hidden={block.divider === false}></div>

{:else if block.type === 'gallery'}
    <div class="gallery">
        {#each (block.items || []).filter((it) => it.url) as it}
            <img src={it.url} alt="media"/>
        {/each}
    </div>

{:else if block.type === 'section'}
    <div class="section">
        <div class="txt">{@html md(block.content)}</div>
        {#if block._accessory === 'image' && block._image}
            <img class="thumb" src={block._image} alt="thumbnail"/>
        {:else if block._accessory === 'button'}
            <span class="btn {btnClass(block._button)}">{buttonLabel(block._button)}</span>
        {/if}
    </div>

{:else if block.type === 'buttons'}
    <div class="btnrow">
        {#each block.buttons as b}
            <span class="btn {btnClass(b)}">{buttonLabel(b)}</span>
        {/each}
    </div>

{:else if block.type === 'container'}
    <div class="container" style={block._accentOn ? `border-left-color:${block._accentHex}` : 'border-left-color:transparent'}>
        {#each block.children as child (child._id)}
            <svelte:self block={child} {subPanels}/>
        {/each}
    </div>
{/if}

<script>
    export let block;
    export let subPanels = [];

    function panelLabel(id) {
        const p = subPanels.find((x) => x.id === id);
        return p ? p.label : "Open ticket";
    }
    function buttonLabel(b) {
        const emoji = b.emoji ? b.emoji + " " : "";
        if (b.kind === "ticket") return emoji + (b.label || panelLabel(b.panelId));
        return emoji + (b.label || b.url || "Button");
    }
    // Link buttons are always grey; ticket buttons follow their chosen colour.
    function btnClass(b) {
        if (b.kind === "link") return "grey";
        return { primary: "blurple", secondary: "grey", success: "green", danger: "red" }[b.style] || "blurple";
    }

    function md(text) {
        if (!text) return "";
        const esc = text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
        return esc
            .replace(/^### (.*)$/gm, "<h3>$1</h3>")
            .replace(/^## (.*)$/gm, "<h2>$1</h2>")
            .replace(/^# (.*)$/gm, "<h1>$1</h1>")
            .replace(/^-# (.*)$/gm, '<span class="sub">$1</span>')
            .replace(/\*\*(.*?)\*\*/g, "<strong>$1</strong>")
            .replace(/\*(.*?)\*/g, "<em>$1</em>")
            .replace(/__(.*?)__/g, "<u>$1</u>")
            .replace(/~~(.*?)~~/g, "<s>$1</s>")
            .replace(/`(.*?)`/g, "<code>$1</code>")
            .replace(/\n/g, "<br>");
    }
</script>

<style>
    .txt { font-size: 0.9rem; line-height: 1.4; color: #dbdee1; word-wrap: break-word; }
    .txt :global(h1) { font-size: 1.3rem; margin: 2px 0; }
    .txt :global(h2) { font-size: 1.15rem; margin: 2px 0; }
    .txt :global(h3) { font-size: 1rem; margin: 2px 0; }
    .txt :global(code) { background: rgba(0,0,0,0.3); padding: 1px 4px; border-radius: 3px; font-family: monospace; }
    .txt :global(.sub) { font-size: 0.78rem; color: #949ba4; }

    .sep { height: 1px; background: rgba(255,255,255,0.1); margin: 2px 0; }
    .sep.hidden { background: transparent; }

    .gallery { display: grid; grid-template-columns: repeat(auto-fill, minmax(90px, 1fr)); gap: 6px; }
    .gallery img { width: 100%; height: 90px; object-fit: cover; border-radius: 6px; }

    .section { display: flex; justify-content: space-between; gap: 12px; align-items: center; }
    .section .txt { flex: 1; }
    .thumb { width: 72px; height: 72px; object-fit: cover; border-radius: 6px; flex-shrink: 0; }

    .btnrow { display: flex; flex-wrap: wrap; gap: 6px; }
    .btn { background: #5865f2; color: #fff; border-radius: 4px; padding: 6px 12px; font-size: 0.85rem; white-space: nowrap; }
    .btn.blurple { background: #5865f2; }
    .btn.grey { background: #4e5058; }
    .btn.green { background: #248046; }
    .btn.red { background: #da373c; }

    .container {
        background: #2b2d31; border-left: 4px solid transparent; border-radius: 4px;
        padding: 10px 12px; display: flex; flex-direction: column; gap: 8px; margin: 2px 0;
    }
</style>
