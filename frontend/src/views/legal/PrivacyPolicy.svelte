<script>
    import Card from "../../components/Card.svelte";
</script>

<div class="content">
    <div class="card-wrapper">
        <Card footer={false} fill={false}>
            <span slot="title"> Privacy Policy </span>
            <div slot="body" style="width: 100%">
                <h2>What data do you collect & how do you use the data?</h2>
                <p>
                    When you invite Tickets to your server, we cache the guild
                    object, channel objects and role objects. Additionally, we
                    may also cache any user or member objects that Discord
                    provides Tickets with in the GUILD_CREATE payload. Upon
                    interaction with the bot, we may also cache your member and
                    user objects if we do not already do so. We store this data
                    solely for operation of the Tickets service to prevent
                    excessive API requests to the Discord API.
                </p>
                <br />
                <p>
                    When a Ticket is closed in a server, all message content and
                    participants (specifically username, avatar hash and
                    discriminator) as retrieved and uploaded to our ticket
                    archive store. We use OVH's object storage offering to store
                    the data, which is encrypted by us, and additionally
                    encrypted by OVH at rest. Backups of our database and ticket
                    transcripts are encrypted and stored in Switzerland, via
                    Exoscale.
                </p>
                <br />
                <p>
                    When logging into our web dashboard, we also collect all
                    guild objects for the servers you are a member of. This is
                    so we can verify your ownership / status in the server, and
                    provide a list of servers to manage.
                </p>
                <hr />
                <h2>Third Party Services</h2>
                <h3>Bloxlink</h3>
                <p>
                    If you have opted into using our Bloxlink integration, the
                    IDs of users will be sent to Bloxlink when opening a ticket
                    to retrieve their Roblox user ID. This is then sent to
                    Roblox's own API and exchanged for the full Roblox user
                    object. Only the user's snowflake is shared with Bloxlink.
                </p>
                <p>
                    You can view Bloxlink's own Terms of Service <a
                        href="https://www.iubenda.com/privacy-policy/11935974s"
                        >here</a
                    >.
                </p>
                <h3>User-created Integrations</h3>
                <p>
                    Administrators of Discord servers using our service can
                    install or create custom integrations. This means that upon
                    opening a ticket, your user ID (which is not classed as PII
                    by Discord) is sent to the operators of the service used by
                    the integration. Custom integrations are able to define
                    privacy policies. You should ask the administrators of
                    servers using our service for the privacy policies of
                    integrations they have installed.
                </p>
                <h3>Cloudflare</h3>
                <p>
                    All traffic is routed through Cloudflare, whose service we
                    use for web performance and security.
                </p>
                <p>
                    You can view Cloudflare's privacy policy <a
                        href="https://www.cloudflare.com/privacypolicy/">here</a
                    >.
                </p>
                <h3>Google Analytics</h3>
                <p>
                    We use Google Analytics on some of our web pages to track
                    how we can improve your experience using our sites. Google
                    Analytics is a industry standard tool, used by most
                    websites. All data is anonymised before being forwarded. We
                    do not use this data for advertising purposes.
                </p>
                <p>
                    You can view Google Analytics' terms of service and privacy
                    policy <a
                        href="https://marketingplatform.google.com/about/analytics/terms/us/"
                        >here</a
                    >.
                </p>
                <hr />
                <h2>
                    How can I contact you regarding concerns with the Tickets
                    service?
                </h2>
                <p>
                    We are happy to resolve any data processing of security
                    concerns you may have. The best way to get in contact with
                    us is by joining our Discord support server and opening a
                    Ticket in the #support channel.
                </p>
                <hr />
                <h2>How can I request my data or data removal?</h2>
                <p>
                    Please join our Discord support server and open a Ticket in
                    the #support channel. Unfortunately, we are unable to help
                    you with these queries via external channels, as we would
                    not be able to reliably verify your identity. We will handle
                    your request within 30 days of it being logged.
                </p>
            </div>
        </Card>
    </div>
</div>

<style>
    .content {
        display: flex;
        height: 100%;
        width: 100%;
        justify-content: center;
    }

    h3 {
        font-weight: bold !important;
    }

    .card-wrapper {
        display: block;
        width: 75%;
        margin-top: 5%;
    }

    @media (max-width: 576px) {
        .card-wrapper {
            width: 100%;
        }
    }
</style>
