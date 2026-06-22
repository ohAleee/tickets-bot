<li>
    <div style="width: 100%">
        <button on:click isTrigger="1" class="btn" target="{newWindow === true ? '_blank' : '_self'}" style="width: 100%">
            {#if icon}
                <i class="fas {icon}"/>
            {/if}

            <span>{title}</span>
        </button>

    </div>
</li>

<style>
    button {
        display: flex;

        justify-content: center;
        align-items: center;

        text-align: center;

        color: white;
        background: var(--primary-gradient);
        border: none;
        /*border-color: var(--primary);*/
        /*border-width: 2px;*/
        border-radius: .25rem;
        margin: 0;

        cursor: pointer;
        transition: background-color 150ms ease-in-out, border-color 150ms ease-in-out;
    }

    i {
        width: 20px;
        text-align: center;
    }
</style>

<script>
    export let title;
    export let icon;
    export let newWindow;

</script>