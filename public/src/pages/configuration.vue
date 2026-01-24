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
            <div v-bind:class="{tab: true, 'tab-active': activeTab === 1}" v-on:click="activeTab = 1">Global Entry List</div>
            <div v-bind:class="{tab: true, 'tab-active': activeTab === 2}" v-on:click="activeTab = 2">Global Ban List</div>
        </div>
        
        <div v-show="activeTab === 0">
            ...
        </div>

        <div v-show="activeTab === 1">
            <globalentrylist />
        </div>

        <div v-show="activeTab === 2">
            <globalbanlist />
        </div>
    </layout>
</template>

<script>
import {layout, end} from "../components";
import {globalentrylist, globalbanlist} from "./configurations";

export default {
    components: {layout, end, globalbanlist, globalentrylist},
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