import React from 'react';
import { useTranslation } from 'react-i18next';

const IdealForm = () => {
  const { t, i18n } = useTranslation();

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
        },
        body: JSON.stringify({
          language: i18n.language
        }),
      }
    );
    const data = await response.json()
    window.location = data.issuer_authentication_url
  };

  return (
    <>
      <form id="container" onSubmit={handleSubmit}>
        <header><h1>{t('add_iban')}</h1></header>
        <main>
          <div id="ideal-form">
            <p>{t('initiate')}</p>
            <p>{t('multiple')}</p>

            <label htmlFor="ideal-bank-element"><p>{t('amount')}</p></label>
            <p>€ 0,01<span className='details'>*</span></p>

            <p className='details'>*{t('minimum')}</p>

          </div>
        </main>
        <footer>
          <div className="actions">
            <div></div>
            <button id="submit-button" type="submit">{t('start')}</button>
          </div>
        </footer>
      </form>
    </>
  );
};

export default IdealForm;
