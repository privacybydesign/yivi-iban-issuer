import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';


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
                    // Point this to your IRMA server:
                    url: 'http://localhost:8088',

                    start: {
                        method: 'POST',
                        body: transactionStatus.jwt,
                        headers: { 'Content-Type': 'text/plain' },
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
                        <p>{transactionStatus?.transaction_status?.name}</p>

                        <label htmlFor="ideal-bank-element">BIC</label>
                        <p>{transactionStatus?.transaction_status?.issuer_id}</p>

                        <label htmlFor="ideal-bank-element">IBAN</label>
                        <p>{transactionStatus?.transaction_status?.iban}</p>

                        <div id="yivi-web-form">
                        </div>
                    </div>
                </main>
                <footer>
                    <div className="actions">
                    <Link to="/" id="back-button">
                        Again
                    </Link>
                        <div></div>
                    </div>
                </footer>
            </div>
        </>
    );
};

export default IssueCredential;
