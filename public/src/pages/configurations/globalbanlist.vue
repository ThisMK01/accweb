<template>
    <div>
        <p>Share a list of banned players across all your servers that have this feature enabled.</p>

        <button v-show="!enabled" v-on:click="toggleEnabled()" class="primary">Enable</button>
        <button v-show="enabled" v-on:click="toggleEnabled()" class="danger">Disable</button>

        <div class="box condensed">
            <p>Add new entry</p>

            <div class="server-settings-container three-columns">
                <field type="text" :label="$t('player_name_label')" v-model="playerName"></field>
                <field type="text" :label="$t('player_id_label')" v-model="playerId"></field>
                <button v-on:click="addEntry()" class="">{{$t("add_new_button")}}</button>
            </div>
        </div>

        <div>
            <h3>Entries</h3>

            <table>
                <tr>
                    <th style="min-width: 170px;">Player Name</th>
                    <th style="min-width: 170px;">Player ID</th>
                    <th style="min-width: 20px;"></th>
                </tr>

                <tr v-for="entry, i in entries"
                    :key="i"
                >
                    <td>{{ entry.playerName }}</td>
                    <td>{{ entry.playerId }}</td>
                    <td align="center">
                        <i class="fas fa-trash" v-on:click="removeEntry(i)" :title="$t('delete_entry')"></i>
                    </td>
                </tr>
            </table>
        </div>
    </div>
</template>

<script>
import axios from "axios";
import field from "../../components/field.vue";

export default {
    components: {field},
    data() {
        return {
            playerName: "",
            playerId: "",
            entryIndex: 0,
            entries: [],
            enabled: false,
        };
    },
    mounted() {
        this.loadList();
    },
    methods: {
        loadList() {
            axios.get(`/api/configure/global-ban`)
                .then(r => {
                    this.entries = r.data.entries || [];
                    this.enabled = r.data.enabled;
                })
                .catch(e => {
                    console.log(e);
                    this.$store.commit("toast", this.$t("load_entries_error"))
                });
        },

        addEntry() {
            const data = {
                playerId: this.playerId,
                playerName: this.playerName
            }

            axios.post(`/api/configure/global-ban`, data)
                .then(r => {
                    this.loadList();

                    this.playerId = "";
                    this.playerName = "";
                })
                .catch(e => {
                    console.log(e.response.data);
                    this.$store.commit("toast", this.$t("add_entry_error", {error: e.response.data.error}))
                });
        },

        removeEntry(id) {
            if (!window.confirm(this.$t("confirm_remove_entry"))) {
                return;
            }

            axios.delete(`/api/configure/global-ban/${id}`)
                .then(r => {
                    this.loadList();
                })
                .catch(e => {
                    console.log(e);
                    this.$store.commit("toast", this.$t("delete_entry_error"))
                });
        },

        toggleEnabled() {
            axios.post(`/api/configure/global-ban/enable-toggle`)
                .then(r => {
                    this.loadList();
                })
                .catch(e => {
                    console.log(e.response.data);
                    this.$store.commit("toast", this.$t("enable_toggle_error", {error: e.response.data.error}))
                });
        }
    }
}
</script>

<style scoped>

th {
    background-color: #1b2838;
}

td,
th {
    text-align: left;
    padding: 5px;
}

tr:nth-child(odd) {
    background-color: #1f2936;
}

</style>

<i18n>
{
    "en": {
        "player_name_label": "Player Name",
        "player_id_label": "PlayerID",
        "add_new_button": "Add New ban",
        "load_entries_error": "Failed to load ban entries",
        "delete_entry_error": "Failed to delete ban entry",
        "add_entry_error": "Failed to add new ban entry: {error}",
        "delete_entry": "Delete ban entry",
        "confirm_remove_entry": "Do you really want to remove this ban entry?"
    }
}
</i18n>
