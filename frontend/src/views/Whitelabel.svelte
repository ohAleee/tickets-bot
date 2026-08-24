<script>
    import {
        notifyError,
        notifyRatelimit,
        notifySuccess,
        withLoadingScreen,
    } from "../js/util";
    import axios from "axios";
    import Card from "../components/Card.svelte";
    import Button from "../components/Button.svelte";
    import { API_URL } from "../js/constants";
    import { setDefaultHeaders } from "../includes/Auth.svelte";
    import Dropdown from "../components/form/Dropdown.svelte";
    import Input from "../components/form/Input.svelte";

    setDefaultHeaders();

    let token;
    let bots = [];
    let servers = [];
    let errors = [];

    // The status and emoji forms act on one bot at a time.
    let selectedBotId;
    let status = { status: "", status_type: "0" };
    let emojiNames = [];
    let emojis = {};

    $: selectedBot = bots.find((bot) => bot.id === selectedBotId);

    function botName(id) {
        const bot = bots.find((bot) => bot.id === id);
        return bot === undefined ? id : bot.username;
    }

    function invite(botId) {
        window.open(
            "https://discord.com/oauth2/authorize?client_id=" +
                botId +
                "&scope=bot+applications.commands&permissions=805825784",
            "_blank",
        );
    }

    async function submitToken() {
        const res = await axios.post(`${API_URL}/user/whitelabel/bots`, {
            token: token,
        });
        if (res.status !== 200 || !res.data.success) {
            notifyError(res.data);
            return;
        }

        token = "";

        await loadBots();
        await loadServers();
        notifySuccess(`Started tickets whitelabel on ${res.data.username}`);
    }

    async function loadBots() {
        const res = await axios.get(`${API_URL}/user/whitelabel/bots`);
        if (res.status !== 200) {
            if (res.status !== 404) {
                notifyError(res.data);
            }
            return;
        }

        bots = res.data || [];

        if (bots.length > 0 && !bots.some((bot) => bot.id === selectedBotId)) {
            await selectBot(bots[0].id);
        }
    }

    async function selectBot(botId) {
        selectedBotId = botId;

        const bot = bots.find((bot) => bot.id === botId);
        status = {
            status: bot?.status ?? "",
            status_type: bot?.status_type ?? "0",
        };

        await loadEmojis();
    }

    async function loadServers() {
        const res = await axios.get(`${API_URL}/user/whitelabel/servers`);
        if (res.status !== 200) {
            notifyError(res.data);
            return;
        }

        servers = res.data || [];
    }

    async function assignBot(server, botId) {
        const res = await axios.put(
            `${API_URL}/user/whitelabel/servers/${server.guild_id}`,
            { bot_id: botId },
        );
        if (res.status !== 200 || !res.data.success) {
            notifyError(res.data);
            // Put the dropdown back on the bot that is actually serving the guild.
            await loadServers();
            return;
        }

        notifySuccess(
            `${botName(botId)} now answers tickets in ${server.name}. Panels posted by the previous bot must be re-sent.`,
        );
        await loadServers();
    }

    async function loadEmojis() {
        const res = await axios.get(
            `${API_URL}/user/whitelabel/bots/${selectedBotId}/emojis`,
        );
        if (res.status !== 200) {
            notifyError(res.data);
            return;
        }

        emojiNames = res.data.names || [];
        emojis = buildEmojiForm(res.data.emojis);
    }

    // The API omits unset slots; the form needs an entry for every name.
    function buildEmojiForm(stored) {
        const form = {};
        for (const name of emojiNames) {
            form[name] = stored?.[name]?.emoji_id ?? "";
        }
        return form;
    }

    async function saveEmojis() {
        const body = {};
        for (const [name, id] of Object.entries(emojis)) {
            body[name] = { emoji_id: `${id}`.trim() || "0", animated: false };
        }

        const res = await axios.put(
            `${API_URL}/user/whitelabel/bots/${selectedBotId}/emojis`,
            body,
        );
        if (res.status !== 200 || !res.data.success) {
            notifyError(res.data);
            return;
        }

        notifySuccess("Emojis saved");
    }

    async function importEmojis() {
        const res = await axios.post(
            `${API_URL}/user/whitelabel/bots/${selectedBotId}/emojis/import`,
        );
        if (res.status !== 200) {
            notifyError(res.data);
            return;
        }

        emojiNames = res.data.names || [];
        emojis = buildEmojiForm(res.data.emojis);

        const count = Object.keys(res.data.emojis || {}).length;
        if (count === 0) {
            notifyError({
                error: "No emojis found on this application. Upload them in the Developer Portal (your app > Emojis), naming them exactly as listed below.",
            });
            return;
        }

        notifySuccess(`Imported ${count} emojis from the application`);
    }

    async function updateStatus() {
        const res = await axios.post(
            `${API_URL}/user/whitelabel/bots/${selectedBotId}/status`,
            status,
        );
        if (res.status !== 200 || !res.data.success) {
            if (res.status === 429) {
                notifyRatelimit();
            } else {
                notifyError(res.data);
            }
            return;
        }

        await loadBots();
        notifySuccess("Updated status successfully");
    }

    async function deleteStatus() {
        const res = await axios.delete(
            `${API_URL}/user/whitelabel/bots/${selectedBotId}/status`,
        );
        if (res.status !== 200 || !res.data.success) {
            if (res.status === 429) {
                notifyRatelimit();
            } else {
                notifyError(res.data);
            }
            return;
        }

        status = { status: "", status_type: "0" };
        await loadBots();
        notifySuccess("Deleted status successfully");
    }

    async function createSlashCommands(botId) {
        const res = await axios.post(
            `${API_URL}/user/whitelabel/bots/${botId}/create-interactions`,
            {},
            { timeout: 20 * 1000 },
        );
        if (res.status !== 200 || !res.data.success) {
            notifyError(res.data);
            return;
        }

        notifySuccess(
            "Slash commands have been created. Please note, they may take a few minutes before they are visible.",
        );
    }

    async function resync(botId) {
        const res = await axios.post(
            `${API_URL}/user/whitelabel/bots/${botId}/resync`,
            {},
            { timeout: 30 * 1000 },
        );
        if (res.status !== 200 || !res.data.success) {
            notifyError(res.data);
            return;
        }

        await loadServers();
        notifySuccess("Bot resynced");
    }

    async function deleteBot(botId) {
        const res = await axios.delete(
            `${API_URL}/user/whitelabel/bots/${botId}`,
        );
        if (res.status !== 204) {
            notifyError(res.data);
            return;
        }

        selectedBotId = undefined;
        await loadBots();
        await loadServers();
        notifySuccess("Bot deleted");
    }

    async function loadErrors() {
        const res = await axios.get(`${API_URL}/user/whitelabel/errors`);
        if (res.status !== 200 || !res.data.success) {
            notifyError(res.data);
            return;
        }

        if (res.data.errors !== null) {
            errors = res.data.errors.map((error) =>
                Object.assign({}, error, { time: new Date(error.time) }),
            );
        }
    }

    withLoadingScreen(async () => {
        await loadBots();
        await Promise.all([loadServers(), loadErrors()]);
    });
</script>

<div class="wrapper">
    <div class="col">
        <Card footer={false} fill={false}>
            <h4 slot="title">Add a Bot</h4>
            <div slot="body" class="full-width">
                <form class="full-width" on:submit|preventDefault>
                    <label class="form-label">Bot Token</label>

                    <input
                        name="token"
                        type="text"
                        bind:value={token}
                        class="form-input full-width"
                        placeholder="xxxxxxxxxxxxxxxxxxxxxxxx.xxxxxx.xxxxxxxxxxxxxxxxxxxxxxxxxxx"
                    />
                    <p>
                        Note: You will not be able to view the token after
                        submitting it. You can add as many bots as you like.
                    </p>

                    <div class="buttons">
                        <Button
                            icon="fas fa-paper-plane"
                            on:click={submitToken}
                            fullWidth={true}
                            >Submit
                        </Button>
                    </div>
                </form>
            </div>
        </Card>

        {#if bots.length > 0}
            <Card footer={false} fill={false}>
                <h4 slot="title">Bots</h4>
                <div slot="body" class="full-width">
                    {#each bots as bot}
                        <div class="bot-row">
                            <div>
                                <b>{bot.username}</b>
                                <span class="muted"
                                    >{bot.id} · {bot.guild_count} server{bot.guild_count ===
                                    1
                                        ? ""
                                        : "s"}</span
                                >
                            </div>

                            <div class="buttons">
                                <Button
                                    icon="fas fa-plus"
                                    on:click={() => invite(bot.id)}
                                >
                                    Invite
                                </Button>

                                <Button
                                    icon="fas fa-paper-plane"
                                    on:click={() => createSlashCommands(bot.id)}
                                >
                                    Slash Commands
                                </Button>

                                <Button
                                    icon="fas fa-rotate"
                                    on:click={() => resync(bot.id)}
                                >
                                    Resync
                                </Button>

                                <Button
                                    icon="fas fa-trash-can"
                                    on:click={() => deleteBot(bot.id)}
                                    danger
                                >
                                    Delete
                                </Button>
                            </div>
                        </div>
                    {/each}
                </div>
            </Card>

            <Card footer={false} fill={false}>
                <h4 slot="title">Servers</h4>
                <div slot="body" class="full-width">
                    {#if servers.length === 0}
                        <p>
                            None of your bots are in a server yet - use Invite
                            above.
                        </p>
                    {:else}
                        <p>
                            Choose which bot answers tickets in each server. The
                            other bots stay in the server but go idle there, and
                            panels posted by the previous bot must be re-sent.
                        </p>

                        <table class="server-table">
                            <thead>
                                <tr style="border-bottom: 1px solid #dee2e6;">
                                    <th class="table-col">Server</th>
                                    <th class="table-col">Served by</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each servers as server}
                                    <tr class="table-row table-border">
                                        <td class="table-col">{server.name}</td>
                                        <td class="table-col">
                                            <select
                                                class="form-input full-width"
                                                value={server.assigned_bot_id}
                                                on:change={(e) =>
                                                    assignBot(
                                                        server,
                                                        e.target.value,
                                                    )}
                                            >
                                                {#each server.present_bot_ids as botId}
                                                    <option value={botId}
                                                        >{botName(botId)}</option
                                                    >
                                                {/each}
                                            </select>
                                        </td>
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    {/if}
                </div>
            </Card>
        {/if}
    </div>

    <div class="col">
        {#if bots.length > 0}
            <Card footer={false} fill={false}>
                <h4 slot="title">Bot Settings</h4>
                <div slot="body" class="full-width">
                    <Dropdown
                        col1
                        label="Bot"
                        value={selectedBotId}
                        on:change={(e) => selectBot(e.target.value)}
                    >
                        {#each bots as bot}
                            <option value={bot.id}>{bot.username}</option>
                        {/each}
                    </Dropdown>
                </div>
            </Card>

            <Card footer={false} fill={false}>
                <h4 slot="title">Custom Status</h4>
                <div slot="body" class="full-width">
                    <form
                        class="form-wrapper full-width"
                        on:submit|preventDefault
                    >
                        <div class="row">
                            <Dropdown
                                col3
                                label="Status Type"
                                bind:value={status.status_type}
                            >
                                <option value="0">Playing</option>
                                <option value="2">Listening</option>
                                <option value="3">Watching</option>
                                <option value="5">Competing</option>
                                <option value="4">Custom</option>
                            </Dropdown>

                            <div class="col-2-3">
                                <Input
                                    col1
                                    label="Status Text"
                                    placeholder="/help"
                                    bind:value={status.status}
                                />
                            </div>
                        </div>

                        <div class="buttons">
                            <Button
                                icon="fas fa-paper-plane"
                                on:click={updateStatus}
                                fullWidth={true}
                            >
                                Submit
                            </Button>
                            {#if selectedBot?.status}
                                <Button
                                    icon="fas fa-trash-can"
                                    on:click={deleteStatus}
                                    danger
                                    fullWidth={true}
                                >
                                    Clear Status
                                </Button>
                            {/if}
                        </div>
                    </form>
                </div>
            </Card>

            <Card footer={false} fill={false}>
                <h4 slot="title">Emojis</h4>
                <div slot="body" class="full-width">
                    <p>
                        Discord only lets an application use the emojis it owns,
                        so this bot cannot render the main bot's emojis. Upload
                        them to <b>this</b> application (Developer Portal &gt; your
                        app &gt; Emojis), then import them or paste their ids below.
                        Slots you leave empty render without an emoji.
                    </p>

                    <div class="buttons">
                        <Button icon="fas fa-download" on:click={importEmojis}>
                            Import from application
                        </Button>
                    </div>

                    <form
                        class="form-wrapper full-width"
                        on:submit|preventDefault
                    >
                        {#each emojiNames as name}
                            <Input
                                col1
                                label={name}
                                placeholder="Emoji ID"
                                bind:value={emojis[name]}
                            />
                        {/each}

                        <div class="buttons">
                            <Button
                                icon="fas fa-paper-plane"
                                on:click={saveEmojis}
                                fullWidth={true}
                            >
                                Save Emojis
                            </Button>
                        </div>
                    </form>
                </div>
            </Card>
        {/if}

        <Card footer={false} fill={false}>
            <h4 slot="title">Error Log</h4>
            <div slot="body" class="full-width">
                <table class="error-log">
                    <thead>
                        <tr style="border-bottom: 1px solid #dee2e6;">
                            <th class="table-col">Bot</th>
                            <th class="table-col">Error</th>
                            <th class="table-col">Time</th>
                        </tr>
                    </thead>
                    <tbody id="error_body">
                        {#each errors as error}
                            <tr class="table-row table-border">
                                <td class="table-col"
                                    >{error.bot_id
                                        ? botName(error.bot_id)
                                        : "-"}</td
                                >
                                <td class="table-col">{error.message}</td>
                                <td class="table-col"
                                    >{error.time.toLocaleString()}</td
                                >
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        </Card>
    </div>
</div>

<style>
    .wrapper {
        display: flex;
        flex-direction: row;
        height: 100%;
        width: 100%;
        padding: 2%;
        gap: 2%;
    }

    .col {
        display: flex;
        flex-direction: column;
        width: 50%;
        gap: 2%;
    }

    @media only screen and (max-width: 1180px) {
        .wrapper {
            flex-direction: column;
        }

        .col {
            width: 100%;
        }
    }

    /* TODO: Move to central stylesheet*/
    :global(.form-label) {
        font-size: 12px;
        margin-bottom: 5px;
        color: #9a9a9a;
        text-transform: uppercase;
    }

    :global(.form-input),
    :global(.form-input:focus-visible) {
        border-color: #262b3d !important;
        background-color: #262b3d !important;
        color: white !important;
        outline: none;
        padding: 8px 12px;
        margin: 0 0 0.5em 0;
        height: 48px;
    }

    .full-width {
        width: 100%;
    }

    .buttons {
        display: flex;
        flex-direction: row;
        gap: 12px;
        margin-top: 12px;
        flex-wrap: wrap;
    }

    @media only screen and (max-width: 576px) {
        .buttons {
            flex-direction: column;
        }
    }

    .bot-row {
        display: flex;
        flex-direction: column;
        padding: 10px 0;
        border-top: 1px solid #dee2e6;
    }

    .bot-row:first-child {
        border-top: none;
    }

    .muted {
        color: #9a9a9a;
        margin-left: 8px;
        font-size: 13px;
    }

    .error-log,
    .server-table {
        width: 100%;
        border-collapse: collapse;
    }

    .table-col {
        text-align: left;
        padding: 5px 10px;
    }

    .table-border {
        border-top: 1px solid #dee2e6;
    }

    .form-wrapper {
        display: flex;
        flex-direction: column;
    }

    .row {
        display: flex;
        flex-direction: row;
        width: 100%;
        gap: 2%;
    }

    .col-2-3 {
        width: 66%;
    }

    @media only screen and (max-width: 950px) {
        .row {
            flex-direction: column;
        }

        .col-2-3 {
            width: 100%;
        }
    }
</style>
