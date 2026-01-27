// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {useSelector} from 'react-redux';
import {IntlProvider} from 'react-intl';
import {getCurrentUser} from 'mattermost-redux/selectors/entities/users';

import {getMessages} from '../i18n';

interface Props {
    children: React.ReactNode;
}

const IntlProviderWrapper: React.FC<Props> = ({children}) => {
    const currentUser = useSelector(getCurrentUser);

    // Mattermost 사용자 언어 설정 사용 (기본값: ko)
    const mattermostLocale = currentUser?.locale || 'ko';
    const language = mattermostLocale.split('-')[0];

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
