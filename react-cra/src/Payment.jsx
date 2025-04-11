import {useEffect, useState} from 'react';

import {Elements} from '@stripe/react-stripe-js';
import CheckoutForm from './CheckoutForm'

function Payment(props) {
  const { stripePromise } = props;
  const [ clientSecret, setClientSecret ] = useState('');

  useEffect(() => {
    // Create PaymentIntent as soon as the page loads
    fetch("/api/create-payment-intent")
      .then((res) => res.json())
      .then(({clientSecret}) => setClientSecret(clientSecret));
  }, []);


  return (
    <>
      {clientSecret && stripePromise && (
        <div>
          <h1>test</h1>
        <Elements stripe={stripePromise} options={{ clientSecret, }}>
          <CheckoutForm />
        </Elements>
        </div>
      )}
    </>
  );
}

export default Payment;
