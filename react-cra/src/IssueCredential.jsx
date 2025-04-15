import React, { useEffect, useState } from 'react';

const IssueCredential = () => {
    const [transactionStatus, setTransactionStatus] = useState(null);

    useEffect(() => {
        const fetchTransactionStatus = async () => {
            const params = new URLSearchParams(window.location.search);
            const transactionId = params.get('trxid');

            if (!transactionId) {
                alert('Something went wrong')
            }
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
            setTransactionStatus(data);
        };
        fetchTransactionStatus();
    }, []);

    useEffect(() => {
        import("@privacybydesign/yivi-frontend").then((yivi) => {
            const web = yivi.newWeb({
                debugging: true,
                element: '#yivi-web-form',

                // Back-end options
                session: {
                    // Point this to your controller:
                    url: '/api',

                    start: {
                        url: (o) => `${o.url}/session`,
                        method: 'POST',
                    },
                    result: {
                        url: (o, { sessionToken }) => `${o.url}/token/${sessionToken}`,
                        method: 'GET',
                    }
                }
            });
            web.start()
                .then((result) => {
                    setAccessToken(result.access);
                    router.push('/organizations');
                })
                .catch((err) => {
                    alert(err);
                });
        });
    }, [transactionStatus]);

    return (
        <>
            <div id="container">
                <header><h1>Adding an IBAN</h1></header>
                <main>
                    <div id="ideal-form">
                        <p>The following information was returned from the iDEAL payment.</p>
                        <p>Scan the QR code below to add this information to Yivi..</p>

                        <label htmlFor="ideal-bank-element">Naam</label>
                        <p>{transactionStatus?.name}</p>

                        <label htmlFor="ideal-bank-element">BIC</label>
                        <p>{transactionStatus?.issuer_id}</p>

                        <label htmlFor="ideal-bank-element">IBAN</label>
                        <p>{transactionStatus?.iban}</p>

                        <div id="yivi-web-form">
                        </div>
                    </div>
                </main>
                <footer>
                    <div className="actions">
                        <a id="back-button">
                            Back
                        </a>
                        <div></div>
                    </div>
                </footer>
            </div>
        </>
    );
};

export default IssueCredential;
