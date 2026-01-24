<template>
    <div>
        <p>
            Share a list of administrator entries across all 
            your servers that have this feature enabled.
        </p>

        <checkbox :label="$t('enabled_label')" v-model="enabled"></checkbox>

        <button class="primary" v-on:click="save()"><i class="fas fa-save"></i> {{ $t('save_btn') }}</button>

        <entrylist ref="entrylist" :expanded="true"></entrylist>
    </div>
</template>

<script>
import axios from "axios";
import {layout, end, entrylist, checkbox} from "../../components";

export default {
    components: {layout, end, entrylist, checkbox},
    data() {
        return {
            enabled: false,
        };
    },
    mounted() {
        this.loadEntries();
    },
    methods: {
        loadEntries() {
            axios.get('/api/configure/global-entrylist')
                .then(r => {
                    this.enabled = r.data.enabled;
                    this.$refs.entrylist.setData({entries: r.data.entries});
                })
                .catch(e => {
                    console.log(e);
                    this.$store.commit("toast", "Failed to load global entry list. {err}", {err: e.response.data.error});
                });
        },

        save() {
            const entrylist = this.$refs.entrylist.getData()

            const data = {
                enabled: this.enabled,
                entries: entrylist.entries
            }

            axios.post('/api/configure/global-entrylist', data)
                .then(_ => {
                    this.$store.commit("toast", "Saved!")
                })
                .catch(e => {
                    this.$store.commit("toast", "Failed to save global entry list. {err}", {err: e.response.data.error});
                });

            console.log(data);
        }
    }
}
</script>

<i18n>
{
    "en": {
        "save_btn": "Save List",
        "enabled_label": "Enable Global Entry List"
    }
}
</i18n>
