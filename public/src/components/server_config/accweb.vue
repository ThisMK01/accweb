<template>
    <collapsible :title="$t('title')">
        <div class="server-settings-container two-columns">
            <div>
                <checkbox :label="$t('autostart_label')" v-model="autoStart"></checkbox>
                <checkbox :label="$t('enable_global_entry_label')" v-model="enableGlobalEntrylist"></checkbox>
                <checkbox :label="$t('enable_global_ban_label')" v-model="enableGlobalBanlist"></checkbox>
                
                <div style="margin-top: 15px; border-top: 1px solid #233D51; padding-top: 10px;">
                    <field :label="$t('event_id_label')" :placeholder="$t('event_id_placeholder')" v-model="eventId"></field>
                    <checkbox :label="$t('collector_enabled_label')" v-model="collectorEnabled"></checkbox>
                </div>
            </div>
            
            <div>
                <div v-if="os.name == 'windows'">
                    <checkbox :label="$t('enable_adv_windows_conf')" v-model="enableAdvWindowsCfg"></checkbox>
                </div>
                <div v-if="os.name == 'windows' && enableAdvWindowsCfg" style="padding: 10px;">
                        <div class="alert">{{$t('adv_windows_alert')}}</div>

                        <div class="server-settings-container two-columns">
                            <selection :label="$t('cpu_priority_label')" :options="priorities" v-model="advWindowsCfg.cpuPriority"></selection>

                            <checkbox :label="$t('enable_windows_firewall')" v-model="advWindowsCfg.enableWindowsFirewall"></checkbox>
                        </div>        
            
                        <label>Core Affinity: (Empty means ALL CPUs)</label> <br /> 
                        <div class="server-settings-container four-columns">
                            <checkbox :label="'CPU '+(n-1)" v-for="n in os.numCpu" :key="n" v-model="coreAffinityCPU[n-1]"></checkbox>
                        </div>
                    </div>
            </div>
        </div>
    </collapsible>
</template>

<style>
.alert {
    border: 1px solid #3f0b0b;
    padding: 5px;
    background-color: red;
    font-weight: bold;
    margin-bottom: 10px;
}
</style>

<script>
import collapsible from "../collapsible.vue";
import checkbox from "../checkbox.vue";
import selection from "../selection.vue";
import field from "../field.vue";
import axios from "axios";

export default {
    components: {collapsible, checkbox, selection, field},
    data() {
        return {
            autoStart: false,
            enableAdvWindowsCfg: false,
            enableGlobalEntrylist: false,
            enableGlobalBanlist: false,
            advWindowsCfg: {
                enableWindowsFirewall: false,
                cpuPriority: 32,
                coreAffinity: 0
            },
            priorities: [
                {value: 256, label: "Realtime"},
                {value: 128, label: "High"},
                {value: 32768, label: "Above Normal"},
                {value: 32, label: "Normal"},
                {value: 16384, label: "Below Normal"},
                {value: 64, label: "Low"},
            ],
            coreAffinityCPU: [],
            os: {
                name: "",
                numCpu: 0
            },
            eventId: "",
            collectorEnabled: false,
            collectorStatus: "stopped"
        };
    },
    methods: {
        hasCPUAffinity(n) {
            if (this.advWindowsCfg.coreAffinity === 0) {
                console.log("CPU Affinity was ZERO!");
                this.advWindowsCfg.coreAffinity = Math.pow(2, this.os.numCpu) - 1;
            }

            return this.advWindowsCfg.coreAffinity & Math.pow(2, n) ? true : false;
        },
        calculateAffinity() {
            let total = 0;
            for (const i in this.coreAffinityCPU) {
                if (Object.hasOwnProperty.call(this.coreAffinityCPU, i)) {
                    if (!this.coreAffinityCPU[i]) {
                        continue;
                    }
                    
                    total += Math.pow(2, i);
                }
            }
            return total;
        },
        setData(data) {
            this.autoStart = data.autoStart;
            this.enableAdvWindowsCfg = data.enableAdvWindowsCfg;
            this.enableGlobalEntrylist = data.enableGlobalEntrylist;
            this.enableGlobalBanlist = data.enableGlobalBanlist;
            this.eventId = data.event_id || "";
            this.collectorEnabled = !!data.collector_enabled;
            this.collectorStatus = data.collector_status || (this.collectorEnabled ? "enabled" : "stopped");

            if (data.advWindowsCfg !== null) {
                this.advWindowsCfg = data.advWindowsCfg;
            }

            axios.get("/api/metadata")
                .then(r => {
                    this.os = r.data;

                    for (let i = 0; i <= this.os.numCpu; i++) {
                        this.coreAffinityCPU[i] = this.hasCPUAffinity(i)
                    }
                });
        },
        getData() {
            if (this.enableAdvWindowsCfg) {
                this.advWindowsCfg.coreAffinity = this.calculateAffinity();
                this.advWindowsCfg.cpuPriority = parseInt(this.advWindowsCfg.cpuPriority);
            }

            return {
                autoStart: this.autoStart,
                enableAdvWindowsCfg: this.enableAdvWindowsCfg,
                advWindowsCfg: this.advWindowsCfg,
                enableGlobalEntrylist: this.enableGlobalEntrylist,
                enableGlobalBanlist: this.enableGlobalBanlist,
                event_id: this.eventId,
                collector_enabled: this.collectorEnabled,
                collector_status: this.collectorStatus
            };
        }
    }
}
</script>

<i18n>
{
    "en": {
        "title": "ACC Web configuration",
        "autostart_label": "Server instance auto start.",
        "enable_adv_windows_conf": "Advanced Windows Configurations",
        "cpu_priority_label": "Process priority",
        "enable_windows_firewall": "Enable Windows Firewall",
        "enable_global_entry_label": "Enable global entry list",
        "enable_global_ban_label": "Enable global ban list",
        "event_id_label": "Event ID",
        "event_id_placeholder": "e.g. TEST-EVENT-001",
        "collector_enabled_label": "Enable Telemetry Collector",
        "adv_windows_alert": "CAUTION: If you are not familiarized with this terms, DISABLE this feature!"
    }
}
</i18n>
