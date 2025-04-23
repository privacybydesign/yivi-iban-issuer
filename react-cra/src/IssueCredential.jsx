import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

const IssueCredential = () => {
    const [statusResponse, setStatusResponse] = useState(null);
    const [error, setError] = useState(false);
    const [done, setDone] = useState(false);

    const { t, i18n } = useTranslation();

    useEffect(() => {
        const fetchTransactionStatus = async () => {
            const params = new URLSearchParams(window.location.search);
            const transactionId = params.get('trxid');

            if (transactionId) {
                // Call backend API to start iDEAL flow.
                const response = await fetch(
                    '/api/status',
                    {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify({
                            transaction_id: transactionId,
                        })
                    }
                );
                const data = await response.json();
                setStatusResponse(data);
            }
        };
        fetchTransactionStatus().catch((error) => {
            console.error('Error fetching transaction status:', error);
            setError(true);
        });
    }, []);

    useEffect(() => {
        if (statusResponse) {
            if (statusResponse.transaction_status?.status !== 'success') {
                setError(true);
                return;
            }

            import("@privacybydesign/yivi-frontend").then((yivi) => {
                const web = yivi.newWeb({
                    debugging: true,
                    language: i18n.language,
                    element: '#yivi-web-form',

                    // Back-end options
                    session: {
                        // Point this to your IRMA server:
                        url: 'https://is.staging.yivi.app',

                        start: {
                            method: 'POST',
                            body: statusResponse.jwt,
                            headers: { 'Content-Type': 'text/plain' },
                        }
                    }
                });
                web.start()
                    .then(() => {
                        setDone(true);
                    })
                    .catch((err) => {
                        console.error('Error starting Yivi:', err);
                        setError(true);
                    });
            });
        }

    }, [statusResponse]);

    const showError = () => {
        switch (statusResponse?.transaction_status?.status) {
            case 'failure':
                return <p>{t('failure')}</p>;
            case 'cancelled':
                return <p>{t('cancelled')}</p>;
            default:
                return <p>{t('error')}</p>;
        }
    }

    return (
        <>
            <div id="container">
                <header><h1>{t('add_iban')}</h1></header>
                <main>
                    <div id="ideal-form">
                        {error && (
                                <div className="imageContainer">
                                    <img src="/images/fail.png" alt="error" />
                                    {showError()}
                                </div>
                        )}
                        {!error && !done && (
                            <>
                                <p>{t('information')}</p>
                                <p>{t('qr')}</p>

                                <label htmlFor="ideal-bank-element">{t('name')}</label>
                                <p>{statusResponse?.transaction_status?.name}</p>

                                <label htmlFor="ideal-bank-element">BIC</label>
                                <p>{statusResponse?.transaction_status?.issuer_id}</p>

                                <label htmlFor="ideal-bank-element">IBAN</label>
                                <p>{statusResponse?.transaction_status?.iban}</p>

                                <div id="yivi-web-form">
                                </div>
                            </>
                        )}
                        {done && (
                                <div className="imageContainer">
                                    <img src="/images/done.png" alt="error" />
                                    <p>{t('thank_you')}</p>
                                </div>
                        )}
                    </div>
                </main>
                <footer>
                    <div className="actions">
                        <Link to="/" id="back-button">
                            {t('again')}
                        </Link>
                        <div></div>
                    </div>
                </footer>
            </div>
        </>
    );
};

export default IssueCredential;
