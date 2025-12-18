<template>
    <layout>
        <div class="title">
            <h1>{{$t("title")}}</h1>
            <div class="menu">
                <button class="primary" v-on:click="$router.push('/')"><i class="fas fa-arrow-left"></i>
                    {{ $t("back_btn") }}</button>
            </div>
        </div>

        <div class="tabs">
            <div v-bind:class="{tab: true, 'tab-active': activeTab === 0}" v-on:click="activeTab = 0">General</div>
            <div v-bind:class="{tab: true, 'tab-active': activeTab === 1}" v-on:click="activeTab = 1">Global Admins</div>
            <div v-bind:class="{tab: true, 'tab-active': activeTab === 2}" v-on:click="activeTab = 2">Global Bans</div>
        </div>
        
        <div v-show="activeTab === 0">
            ...
        </div>

        <div v-show="activeTab === 1">
            <p>Share a list of administrator entries across all your servers that have this feature enabled.</p>

            <simpleentrylist ref="globalAdmins" :context="'admin'"></simpleentrylist>
        </div>

        <div v-show="activeTab === 2">
            <p>Share a list of banned players across all your servers that have this feature enabled.</p>

            <simpleentrylist ref="globalBans" :context="'ban'"></simpleentrylist>
        </div>
    </layout>
</template>

<script>
import {layout, end, simpleentrylist} from "../components";

export default {
    components: {layout, end, simpleentrylist},
    data() {
        return {
            activeTab: 1,
        };
    },
    mounted() {
    },
    methods: {}
}
</script>

<i18n>
{
    "en": {
        "title": "Configurations",
        "back_btn": "Back"
    }
}
</i18n>