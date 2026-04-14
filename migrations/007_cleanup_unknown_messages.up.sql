DELETE FROM wz_messages
WHERE msg_type = 'unknown'
  AND (
    raw::text LIKE '%protocolMessage%'
    OR raw::text LIKE '%messageStubType%'
    OR (
      raw::text LIKE '%senderKeyDistributionMessage%'
      AND raw::text NOT LIKE '%conversation%'
      AND raw::text NOT LIKE '%imageMessage%'
      AND raw::text NOT LIKE '%videoMessage%'
      AND raw::text NOT LIKE '%audioMessage%'
      AND raw::text NOT LIKE '%documentMessage%'
    )
  );
