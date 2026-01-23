import React from 'react';
import {createIntl, createIntlCache} from 'react-intl';

import manifest from './manifest';

import Root from './components/root';
import AssigneeModal from './components/assignee_modal';
import SidebarRight from './components/sidebar_right';
import IntlProviderWrapper from './components/intl_provider_wrapper';
import {getMessages, getCurrentLanguage} from './i18n';

import {openAddCard, setShowRHSAction, telemetry, updateConfig, setHideTeamSidebar, fetchAllIssueLists} from './actions';
import reducer from './reducer';
import PostTypeTodo from './components/post_type_todo';
import TeamSidebar from './components/team_sidebar';
import ChannelHeaderButton from './components/channel_header_button';
import {getPluginServerRoute} from './selectors';

let activityFunc;
let lastActivityTime = Number.MAX_SAFE_INTEGER;
const activityTimeout = 60 * 60 * 1000; // 1 hour
const {id: pluginId} = manifest;

// IntlProvider로 래핑된 SidebarRight 컴포넌트
const WrappedSidebarRight = (props) => (
    <IntlProviderWrapper>
        <SidebarRight {...props} />
    </IntlProviderWrapper>
);

export default class Plugin {
    initialize(registry, store) {
        // intl 객체 생성 (메뉴 텍스트 등에 사용)
        const cache = createIntlCache();
        const intl = createIntl({
            locale: getCurrentLanguage(),
            messages: getMessages(getCurrentLanguage()),
        }, cache);

        const {toggleRHSPlugin, showRHSPlugin} = registry.registerRightHandSidebarComponent(WrappedSidebarRight, intl.formatMessage({id: 'SidebarRight.listHeading.myTodos', defaultMessage: 'Todo List'}));

        registry.registerReducer(reducer);
        registry.registerRootComponent(Root);
        registry.registerRootComponent(AssigneeModal);

        registry.registerBottomTeamSidebarComponent(TeamSidebar);

        registry.registerPostDropdownMenuAction(
            intl.formatMessage({id: 'Plugin.postMenu.addTodo', defaultMessage: 'Add Todo'}),
            (postID) => {
                telemetry('post_action_click');
                store.dispatch(openAddCard(postID));
                store.dispatch(showRHSPlugin);
            },
        );

        store.dispatch(setShowRHSAction(() => store.dispatch(showRHSPlugin)));
        registry.registerChannelHeaderButtonAction(
            <ChannelHeaderButton/>,
            () => {
                telemetry('channel_header_click');
                store.dispatch(toggleRHSPlugin);
            },
            intl.formatMessage({id: 'Plugin.channelHeader.todo', defaultMessage: 'Todo'}),
            intl.formatMessage({id: 'Plugin.channelHeader.tooltip', defaultMessage: 'Open your list of Todo issues'}),
        );

        const iconURL = getPluginServerRoute(store.getState()) + '/public/app-bar-icon.png';
        registry.registerAppBarComponent(
            iconURL,
            () => store.dispatch(toggleRHSPlugin),
            intl.formatMessage({id: 'Plugin.channelHeader.tooltip', defaultMessage: 'Open your list of Todo issues'}),
        );

        const refresh = () => {
            store.dispatch(fetchAllIssueLists());
        };
        registry.registerWebSocketEventHandler(`custom_${pluginId}_refresh`, refresh);
        registry.registerReconnectHandler(refresh);

        store.dispatch(fetchAllIssueLists(true));

        // register websocket event to track config changes
        const configUpdate = ({data}) => {
            store.dispatch(setHideTeamSidebar(data.hide_team_sidebar));
        };

        registry.registerWebSocketEventHandler(`custom_${pluginId}_config_update`, configUpdate);

        store.dispatch(updateConfig());

        activityFunc = () => {
            const now = new Date().getTime();
            if (now - lastActivityTime > activityTimeout) {
                store.dispatch(fetchAllIssueLists(true));
            }
            lastActivityTime = now;
        };

        document.addEventListener('click', activityFunc);

        registry.registerPostTypeComponent('custom_todo', PostTypeTodo);
    }

    deinitialize() {
        document.removeEventListener('click', activityFunc);
    }
}

window.registerPlugin(pluginId, new Plugin());
