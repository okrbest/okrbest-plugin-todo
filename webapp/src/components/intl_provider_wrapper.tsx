// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {useSelector} from 'react-redux';
import {IntlProvider} from 'react-intl';
import {getCurrentUser} from 'mattermost-redux/selectors/entities/users';

import {getMessages} from '../i18n';
import {UserSettings} from '../userSettings';

interface Props {
    children: React.ReactNode;
}

const IntlProviderWrapper: React.FC<Props> = ({children}) => {
    const currentUser = useSelector(getCurrentUser);

    // 1순위: localStorage 'language'
    // 2순위: Mattermost 사용자 언어 설정
    // 3순위: 기본값 'en'
    let language = UserSettings.language;
    if (!language) {
        const mattermostLocale = currentUser?.locale || 'en';
        language = mattermostLocale.split('-')[0];
    }

    return (
        <IntlProvider
            locale={language}
            messages={getMessages(language)}
        >
            {children}
        </IntlProvider>
    );
};

export default IntlProviderWrapper;
