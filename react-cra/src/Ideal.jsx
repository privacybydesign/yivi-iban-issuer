import React from 'react';

const IdealForm = () => {

  const handleSubmit = async (e) => {
    // We don't want to let default form submission happen here,
    // which would refresh the page.
    e.preventDefault();

    // Call backend API to start iDEAL flow.
    const response = await fetch(
      '/api/ibancheck',
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        }
      }
    );
    const data = await response.json()
    window.location = data.issuer_authentication_url
  };

  return (
    <>
      <form id="container" onSubmit={handleSubmit}>
        <header><h1>Adding an IBAN</h1></header>
        <main>
          <div id="ideal-form">
            <p>This page will initiate an IDEAL payment of €0,01, with that Yivi will verify your IBAN.</p>
            <p>
              Do you want to add multiple IBANs? Then perform these steps multiple times.</p>
            <p>
              Yivi uses CM's IBAN Verification method, more information about this can be found here.
            </p>

            <label htmlFor="ideal-bank-element">Amount</label>
            <p>€ 0,01<span className='details'>*</span></p>

            <p className='details'>*10 cents is the minimum amount we can charge.</p>

          </div>
        </main>
        <footer>
          <div className="actions">
            <div></div>
            <button id="submit-button" type="submit">Start IBAN verification</button>
          </div>
        </footer>
      </form>
    </>
  );
};

export default IdealForm;
