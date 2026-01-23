// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {IntlProvider} from 'react-intl';

import {getMessages, getCurrentLanguage} from '../i18n';

interface Props {
    children: React.ReactNode;
}

const IntlProviderWrapper: React.FC<Props> = ({children}) => {
    const language = getCurrentLanguage();

    return (
        <IntlProvider
            locale={language.split(/[_]/)[0]}
            messages={getMessages(language)}
        >
            {children}
        </IntlProvider>
    );
};

export default IntlProviderWrapper;
