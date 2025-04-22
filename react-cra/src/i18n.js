import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

i18n
    .use(LanguageDetector)
    .use(initReactI18next).init({
        detection: {
            order: ['path', 'navigator'],
            lookupFromPathIndex: 0
        },
        resources: {
            en: {
                translation: {
                    add_iban: "Adding an IBAN to Yivi.",
                    initiate: "This page will initiate an IDEAL payment of €0,01, with that Yivi will verify your IBAN.",
                    multiple: "Do you want to add multiple IBANs? Then perform these steps multiple times.",
                    cm: "Yivi uses CM's IBAN Verification method, more information about this can be found here.",
                    minimum: "1 cent is the minimum amount we can charge",
                    amount: "Amount",
                    start: "Start IBAN verification",

                    failure: "An error occured while processing your payment, please try again.",
                    cancelled: "You cancelled the iDEAL transaction, please try again.",
                    error: "Something went wrong. Please try again.",
                    name: "Name",
                    again: "Again",

                    information: "The following information was returned from the iDEAL payment.",
                    qr: "Scan the QR code below to add this information to Yivi.",

                    thank_you: "Thank you for using Yivi, you can close this page now.",

                }
            },
            nl: {
                translation: {
                    add_iban: "IBAN toevoegen aan Yivi.",
                    initiate: "Deze pagina zal een IDEAL-betaling van €0,01 initiëren, waarmee Yivi uw IBAN zal verifiëren.",
                    multiple: "Wilt u meerdere IBAN's toevoegen? Voer deze stappen dan meerdere keren uit.",
                    cm: "Yivi maakt gebruik van de IBAN-verificatiemethode van CM. Meer informatie hierover vindt u hier.",
                    minimum: "1 cent is het minimumbedrag dat wij in rekening kunnen brengen",
                    amount: "Bedrag",
                    start: "Start IBAN verificatie",

                    failure: "Er is een fout opgetreden bij het verwerken van uw betaling. Probeer het opnieuw.",
                    cancelled: "U heeft de iDEAL-transactie geannuleerd. Probeer het opnieuw.",
                    error: "Er is iets misgegaan. Probeer het opnieuw.",
                    name: "Naam",
                    again: "Opnieuw",

                    information: "De volgende informatie werd geretourneerd vanuit de iDEAL-betaling.",
                    qr: "Scan de QR-code hieronder om deze informatie aan Yivi toe te voegen.",

                    thank_you: "Bedankt voor het gebruik van Yivi, u kunt deze pagina nu sluiten.",
                }
            }
        },
        lng: 'nl', // default language
        fallbackLng: 'en',

        interpolation: {
            escapeValue: false, // react already escapes
        }
    });

export default i18n;