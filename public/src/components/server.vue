<template>
    <div class="server" v-bind:class="{running: formattedServerClientCount > 0}">
        <div class="content">
            <div class="name">
                {{server.name}}
                <div class="actions">
                    <span v-if="is_ro">
                        <i class="fas fa-tv" v-if="server.pid" v-on:click="live" :title="$t('view_live')"></i>
                    </span>
                    <span v-if="!ro">
                        <i class="fas fa-cog" v-on:click="edit" :title="$t('change_config')"></i>
                        <i class="fas fa-terminal" v-on:click="logs" :title="$t('view_logs')"></i>
                        <i class="fas fa-copy" v-on:click="copyConfig" v-if="is_admin" :title="$t('copy_config')"></i>
                        <i class="fas fa-file-download" v-on:click="exportConfig" :title="$t('export_config')"></i>
                        <i class="fas fa-trash" v-on:click="deleteServer" v-if="is_admin" :title="$t('delete_server')"></i>
                    </span>
                </div>
            </div>
            <div class="info">
                <span v-if="server.pid"><b>PID:</b> {{server.pid}} &nbsp;&bull;&nbsp;</span>
                <b>UDP:</b> {{server.udpPort}} &nbsp;&bull;&nbsp;
                <b>TCP:</b> {{server.tcpPort}} &nbsp;&bull;&nbsp;
                <b>{{$t("track")}}:</b> {{server.track}}
                <span v-if="!ro">&nbsp;&bull;&nbsp; <b>{{$t("configuration_directory")}}:</b> {{server.id}}</span>
            </div>
            <div class="info state" v-if="server.pid">
                <b>{{$t("state")}}: </b>{{$t(server.serverState)}} &nbsp;&bull;&nbsp;
                <b>{{$t("number_of_drivers")}}: </b>{{formattedServerClientCount}} &nbsp;&bull;&nbsp;
                <b>{{$t("session")}}: </b>
                <span v-if="server.sessionType">{{server.sessionType}} ({{server.sessionPhase}}) - {{server.sessionRemaining}} min(s)</span>
                <span v-else>{{$t('not_detected')}}</span>
            </div>

            <!-- Collector Section -->
            <div class="collector-section">
                <div class="collector-row">
                    <div class="collector-col-event">
                        <label class="collector-label">{{$t("event_id")}}</label>
                        <input
                            type="text"
                            class="event-id-input"
                            v-model="localEventId"
                            :placeholder="$t('event_id_placeholder')"
                            :disabled="ro || !is_mod || savingCollector || !!server.pid"
                            :title="server.pid ? $t('locked_while_running') : ''"
                            @blur="saveEventId"
                            @keyup.enter="saveEventId"
                        />
                    </div>
                    <div class="collector-col-switch">
                        <label class="collector-label">{{$t("collector")}}</label>
                        <div class="collector-toggle-wrap">
                            <button
                                type="button"
                                class="collector-toggle-btn"
                                :class="localCollectorEnabled ? 'collector-btn-on' : 'collector-btn-off'"
                                :disabled="ro || !is_mod || savingCollector || !!server.pid"
                                :title="server.pid ? $t('locked_while_running') : ''"
                                @click="toggleCollector"
                            >
                                {{ localCollectorEnabled ? $t("collector_on") : $t("collector_off") }}
                            </button>
                            <span class="collector-status-display">
                                <b>{{$t("collector_status_label")}}:</b> {{ formattedStatus }}
                                <span v-if="server.pid" class="locked-hint">({{ $t("locked") }})</span>
                            </span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        <div class="server-actions-side">
            <button class="start" v-on:click="start" v-if="is_mod && !ro && !server.pid">{{$t("start_server")}}</button>
            <button class="stop" v-on:click="stop" v-if="is_mod && !ro && server.pid">{{$t("stop_server")}}</button>
            <div class="online" v-if="ro && server.pid">{{$t("running")}}</div>
            <div class="offline" v-if="ro && !server.pid">{{$t("offline")}}</div>
        </div>
    </div>
</template>

<style scoped>
.content {
    width: 100%;
}

.state {
    margin-top: 10px;
}

.state b {
    color: #505050;
}

.running {
    background-color: #1d2331;
}

.actions {
    display: inline;
    float: right;
    margin-right: 30px;
}

.collector-section {
    margin-top: 12px;
    padding-top: 10px;
    border-top: 1px solid #233D51;
}

.collector-row {
    display: flex;
    flex-wrap: wrap;
    align-items: flex-end;
    gap: 20px;
}

.collector-col-event {
    flex: 1;
    min-width: 200px;
    max-width: 320px;
}

.collector-col-switch {
    display: flex;
    flex-direction: column;
}

.collector-label {
    display: block;
    font-size: 12px;
    color: #c7d5e0;
    margin-bottom: 4px;
    font-weight: bold;
}

.event-id-input {
    width: 100%;
    box-sizing: border-box;
    font-size: 13px;
    background: #1b2838;
    color: #f2f2f2;
    border: 1px solid #233D51;
    border-radius: 2px;
    padding: 6px 10px;
    margin: 0;
}

.event-id-input:focus {
    border-color: #67c1f5;
    outline: none;
}

.event-id-input:disabled {
    background-color: #171a21;
    border-color: #505050;
    color: #888;
}

.collector-toggle-wrap {
    display: flex;
    align-items: center;
    gap: 12px;
}

.collector-toggle-btn {
    padding: 6px 16px;
    font-size: 13px;
    font-weight: bold;
    border-radius: 2px;
    cursor: pointer;
    border: none;
    transition: background 0.15s ease;
    margin: 0;
}

.collector-btn-on {
    background: #4c6b22;
    color: #a4d007;
}

.collector-btn-on:hover:not(:disabled) {
    background: #5d832b;
}

.collector-btn-off {
    background: #233D51;
    color: #c7d5e0;
}

.collector-btn-off:hover:not(:disabled) {
    background: #2d4f69;
}

.collector-toggle-btn:disabled {
    cursor: not-allowed;
    opacity: 0.6;
}

.collector-status-display {
    font-size: 13px;
    color: #c7d5e0;
}

.collector-status-display b {
    color: #505050;
}

.locked-hint {
    color: #888;
    font-size: 11px;
    margin-left: 4px;
}

.server-actions-side {
    display: flex;
    align-items: center;
    margin-left: 15px;
}
</style>

<script>
import axios from "axios";

export default {
    props: ["server", "ro"],
    data() {
        return {
            localEventId: this.server.event_id || "",
            localCollectorEnabled: !!this.server.collector_enabled,
            localCollectorStatus: this.server.collector_status || "stopped",
            savingCollector: false
        };
    },
    watch: {
        "server.event_id"(newVal) {
            this.localEventId = newVal || "";
        },
        "server.collector_enabled"(newVal) {
            this.localCollectorEnabled = !!newVal;
        },
        "server.collector_status"(newVal) {
            this.localCollectorStatus = newVal || (this.localCollectorEnabled ? "enabled" : "stopped");
        }
    },
    computed: {
        formattedServerClientCount: function () {
            return this.server.serverState === 'not_registered' ? '-' : this.server.nrClients;
        },
        formattedStatus() {
            if (this.localCollectorStatus === "enabled" || this.localCollectorEnabled) {
                return this.$t("status_enabled");
            }
            return this.$t("status_stopped");
        }
    },
    methods: {
        edit() {
            this.$router.push(`/server?id=${this.server.id}`);
        },
        logs() {
            this.$router.push(`/logs?id=${this.server.id}`);
        },
        live() {
            this.$router.push(`/live?id=${this.server.id}`);
        },
        copyConfig() {
            axios.post(`/api/instance/${this.server.id}/clone`)
            .then(() => {
                this.$emit("copied");
            })
            .catch(e => {
                this.$store.commit("toast", this.$t("copy_server_error"))
            });
        },
        exportConfig() {
            let link = document.createElement("a");
            link.setAttribute("type", "hidden");
            link.href = `/api/instance/${this.server.id}/export?token=${this.$store.state.auth.token}`;
            document.body.appendChild(link);
            link.click();
            link.remove();
        },
        deleteServer() {
            if (!window.confirm(this.$t("confirm_delete_server"))) {
                return;
            }

            axios.delete(`/api/instance/${this.server.id}`)
            .then(() => {
                this.$emit("deleted");
            })
            .catch(e => {
                this.$store.commit("toast", this.$t("delete_server_error"))
            });
        },
        start() {
            axios.post(`/api/instance/${this.server.id}/start`)
            .then(() => {
                this.$emit("started");
            })
            .catch(e => {
                this.$store.commit("toast", this.$t("start_server_error", {error: e.response.data.error}))
            });
        },
        stop() {
            axios.post(`/api/instance/${this.server.id}/stop`)
            .then(() => {
                this.$emit("stopped");
            })
            .catch(e => {
                this.$store.commit("toast", this.$t("stop_server_error", {error: e.response.data.error}))
            });
        },
        saveEventId() {
            const trimmed = (this.localEventId || "").trim();
            this.localEventId = trimmed;
            if (trimmed === (this.server.event_id || "")) {
                return;
            }
            this.persistCollectorState(trimmed, this.localCollectorEnabled);
        },
        toggleCollector() {
            const nextEnabled = !this.localCollectorEnabled;
            this.localCollectorEnabled = nextEnabled;
            this.persistCollectorState((this.localEventId || "").trim(), nextEnabled);
        },
        persistCollectorState(eventId, enabled) {
            this.savingCollector = true;
            axios.post(`/api/instance/${this.server.id}/collector`, {
                event_id: eventId,
                collector_enabled: enabled
            })
            .then(response => {
                this.savingCollector = false;
                this.localEventId = response.data.event_id;
                this.localCollectorEnabled = response.data.collector_enabled;
                this.localCollectorStatus = response.data.collector_status;
                this.$emit("collector-updated");
            })
            .catch(err => {
                this.savingCollector = false;
                this.localEventId = this.server.event_id || "";
                this.localCollectorEnabled = !!this.server.collector_enabled;
                this.localCollectorStatus = this.server.collector_status || "stopped";
                const msg = (err.response && err.response.data && err.response.data.error) ? err.response.data.error : err.message;
                this.$store.commit("toast", this.$t("collector_update_error", {error: msg}));
            });
        }
    }
}
</script>

<i18n>
{
    "en": {
        "start_server": "Start",
        "stop_server": "Stop",
        "change_config": "Change config",
        "view_logs": "View logs",
        "view_live": "View live",
        "copy_config": "Copy config",
        "export_config": "Export config",
        "delete_server": "Delete server",
        "confirm_delete_server": "Do you really want to delete this server?",
        "copy_server_error": "Error copying server configuration.",
        "delete_server_error": "Error deleting server configuration.",
        "start_server_error": "Error starting server, please check the logs. ERROR: {error}",
        "stop_server_error": "Error stopping server. ERROR: {error}",
        "track": "Track",
        "configuration_directory": "Config dir",
        "running": "Running",

        "state": "State",
        "number_of_drivers": "Drivers",
        "session": "Session",
        "not_detected": "Not detected",

        "offline": "Offline",
        "starting": "Starting",
        "not_registered": "Waiting for events",
        "online": "Online",

        "event_id": "Event ID",
        "event_id_placeholder": "e.g. TEST-EVENT-001",
        "collector": "Collector",
        "collector_on": "ON",
        "collector_off": "OFF",
        "collector_status_label": "Status",
        "status_enabled": "Enabled",
        "status_stopped": "Stopped",
        "locked": "Locked",
        "locked_while_running": "Configuration locked while server is running",
        "collector_update_error": "Error updating collector settings: {error}"
    }
}
</i18n>
