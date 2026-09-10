# WhatsApp Template Message Payloads

Send template messages through `POST /v1/wa/messages`.

The API sets `messaging_product` to `whatsapp`, selects `phone_number_id` from the authenticated user, and defaults `recipient_type` to `individual`. Do not provide those fields.

A template must already be approved in Meta. Only send components and parameters that exist in that approved template. Omit `components` completely when the template has no runtime variables or media.

## Base Payload

```json
{
  "to": "6590073708",
  "type": "template",
  "template": {
    "name": "approved_template_name",
    "language": {
      "code": "en_US"
    },
    "components": []
  }
}
```

| Field | Values / requirements |
| --- | --- |
| `to` | Recipient WhatsApp ID or BSUID, for example `6590000000`. |
| `type` | Always `template`. |
| `template.name` | Exact approved Meta template name. |
| `template.language.code` | Exact template language and locale, for example `en`, `en_US`, or `zh_CN`. |
| `template.language.policy` | Optional legacy policy value. Normally omit it. |
| `template.components` | Omit when no header, body, or button runtime values are needed. |

## Component Types

A send component has this shape:

```json
{
  "type": "body",
  "parameters": []
}
```

| `type` | When to send it | Extra fields |
| --- | --- | --- |
| `header` | The approved header contains a text variable or requires image, video, document, or location media. | `parameters` only. |
| `body` | The approved body contains variables. | `parameters` only. |
| `button` | The approved button has a dynamic URL, quick reply payload, copy code, or flow parameter. | `sub_type`, `index`, and usually `parameters`. |

`footer` is static approved-template content and is never sent as a component.

## Header Payloads

### Text header

Use parameters in the same order as the header variables.

```json
{
  "type": "header",
  "parameters": [
    { "type": "text", "text": "Jane" }
  ]
}
```

### Image header

Use either a Meta media ID or a publicly reachable HTTPS URL, including an OSS URL.

```json
{
  "type": "header",
  "parameters": [
    {
      "type": "image",
      "image": {
        "id": "META_MEDIA_ID"
      }
    }
  ]
}
```

```json
{
  "type": "header",
  "parameters": [
    {
      "type": "image",
      "image": {
        "link": "https://example-bucket.oss-ap-southeast-1.aliyuncs.com/banner.jpg"
      }
    }
  ]
}
```

### Video header

```json
{
  "type": "header",
  "parameters": [
    {
      "type": "video",
      "video": {
        "id": "META_MEDIA_ID"
      }
    }
  ]
}
```

```json
{
  "type": "header",
  "parameters": [
    {
      "type": "video",
      "video": {
        "link": "https://cdn.example.com/intro.mp4"
      }
    }
  ]
}
```

### Document header

`filename` is optional and is shown to the recipient when supplied.

```json
{
  "type": "header",
  "parameters": [
    {
      "type": "document",
      "document": {
        "id": "META_MEDIA_ID",
        "filename": "invoice.pdf"
      }
    }
  ]
}
```

```json
{
  "type": "header",
  "parameters": [
    {
      "type": "document",
      "document": {
        "link": "https://cdn.example.com/invoice.pdf",
        "filename": "invoice.pdf"
      }
    }
  ]
}
```

### Location header

Only use this when the approved template has a location header.

```json
{
  "type": "header",
  "parameters": [
    {
      "type": "location",
      "location": {
        "latitude": 1.3521,
        "longitude": 103.8198,
        "name": "Store name",
        "address": "1 Example Street"
      }
    }
  ]
}
```

## Body Payloads

Parameters replace body variables in their template-defined order. Named variables are also sent by their approved-template order, not by name.

### Text

```json
{
  "type": "body",
  "parameters": [
    { "type": "text", "text": "Jane" },
    { "type": "text", "text": "ORD-12345" }
  ]
}
```

### Currency

`amount_1000` is the amount in thousandths of the major currency unit. For S$25.00, use `25000`.

```json
{
  "type": "body",
  "parameters": [
    {
      "type": "currency",
      "currency": {
        "fallback_value": "S$25.00",
        "code": "SGD",
        "amount_1000": 25000
      }
    }
  ]
}
```

### Date and time

`fallback_value` is required. Meta may use the structured fields for locale-aware rendering.

```json
{
  "type": "body",
  "parameters": [
    {
      "type": "date_time",
      "date_time": {
        "fallback_value": "9 September 2026, 10:30 AM",
        "year": 2026,
        "month": 9,
        "day_of_month": 9,
        "hour": 10,
        "minute": 30
      }
    }
  ]
}
```

## Button Payloads

`index` is the zero-based button position in the approved template: `"0"`, `"1"`, and so on. `sub_type` identifies the approved button type.

### Dynamic URL button

Use the `text` parameter only for the variable suffix of a dynamic URL button.

```json
{
  "type": "button",
  "sub_type": "url",
  "index": "0",
  "parameters": [
    { "type": "text", "text": "ORD-12345" }
  ]
}
```

For an approved URL `https://example.com/orders/{{1}}`, this produces `https://example.com/orders/ORD-12345`.

### Quick reply button

`payload` is opaque metadata returned in the recipient's webhook reply. It is not displayed on the button.

```json
{
  "type": "button",
  "sub_type": "quick_reply",
  "index": "0",
  "parameters": [
    { "type": "payload", "payload": "CONFIRM_ORDER:ORD-12345" }
  ]
}
```

### Copy-code button

Only use this for an approved authentication template with a copy-code button.

```json
{
  "type": "button",
  "sub_type": "copy_code",
  "index": "0",
  "parameters": [
    { "type": "coupon_code", "coupon_code": "SAVE20" }
  ]
}
```

### Flow button

Only use this for an approved template configured with a Flow button. The values must match that Flow's configuration.

```json
{
  "type": "button",
  "sub_type": "flow",
  "index": "0",
  "parameters": [
    {
      "type": "action",
      "action": {
        "flow_token": "customer-123",
        "flow_action_data": {
          "order_id": "ORD-12345"
        }
      }
    }
  ]
}
```

### Static buttons

Buttons without runtime values, including standard phone-number buttons, need no component entry. Do not send a `voice_call` component unless Meta's template-specific documentation explicitly requires parameters for that approved template.

## Full Valid Payload Matrix

These are the valid template component shapes Meta accepts for outbound template sends. Use only the combinations that exist in the approved template.

### 1) Template with no runtime values

```json
{
  "to": "6590073708",
  "type": "template",
  "template": {
    "name": "welcome_message",
    "language": {
      "code": "en_US"
    }
  }
}
```

### 2) Header with text variable

```json
{
  "type": "header",
  "parameters": [
    { "type": "text", "text": "Jane" }
  ]
}
```

### 3) Header with image media by ID

```json
{
  "type": "header",
  "parameters": [
    {
      "type": "image",
      "image": {
        "id": "META_MEDIA_ID"
      }
    }
  ]
}
```

### 4) Header with image media by URL

```json
{
  "type": "header",
  "parameters": [
    {
      "type": "image",
      "image": {
        "link": "https://example.com/banner.jpg"
      }
    }
  ]
}
```

### 5) Header with video media by ID

```json
{
  "type": "header",
  "parameters": [
    {
      "type": "video",
      "video": {
        "id": "META_MEDIA_ID"
      }
    }
  ]
}
```

### 6) Header with video media by URL

```json
{
  "type": "header",
  "parameters": [
    {
      "type": "video",
      "video": {
        "link": "https://cdn.example.com/intro.mp4"
      }
    }
  ]
}
```

### 7) Header with document media by ID

```json
{
  "type": "header",
  "parameters": [
    {
      "type": "document",
      "document": {
        "id": "META_MEDIA_ID",
        "filename": "invoice.pdf"
      }
    }
  ]
}
```

### 8) Header with document media by URL

```json
{
  "type": "header",
  "parameters": [
    {
      "type": "document",
      "document": {
        "link": "https://cdn.example.com/invoice.pdf",
        "filename": "invoice.pdf"
      }
    }
  ]
}
```

### 9) Header with location

```json
{
  "type": "header",
  "parameters": [
    {
      "type": "location",
      "location": {
        "latitude": 1.3521,
        "longitude": 103.8198,
        "name": "Store name",
        "address": "1 Example Street"
      }
    }
  ]
}
```

### 10) Body with text parameters

```json
{
  "type": "body",
  "parameters": [
    { "type": "text", "text": "Jane" },
    { "type": "text", "text": "ORD-12345" }
  ]
}
```

### 11) Body with currency parameter

```json
{
  "type": "body",
  "parameters": [
    {
      "type": "currency",
      "currency": {
        "fallback_value": "S$25.00",
        "code": "SGD",
        "amount_1000": 25000
      }
    }
  ]
}
```

### 12) Body with date_time parameter

```json
{
  "type": "body",
  "parameters": [
    {
      "type": "date_time",
      "date_time": {
        "fallback_value": "9 September 2026, 10:30 AM",
        "year": 2026,
        "month": 9,
        "day_of_month": 9,
        "hour": 10,
        "minute": 30
      }
    }
  ]
}
```

### 13) Button with URL parameter

```json
{
  "type": "button",
  "sub_type": "url",
  "index": "0",
  "parameters": [
    { "type": "text", "text": "ORD-12345" }
  ]
}
```

### 14) Button with quick_reply payload

```json
{
  "type": "button",
  "sub_type": "quick_reply",
  "index": "0",
  "parameters": [
    { "type": "payload", "payload": "CONFIRM_ORDER:ORD-12345" }
  ]
}
```

### 15) Button with copy_code payload

```json
{
  "type": "button",
  "sub_type": "copy_code",
  "index": "0",
  "parameters": [
    { "type": "coupon_code", "coupon_code": "SAVE20" }
  ]
}
```

### 16) Button with flow action

```json
{
  "type": "button",
  "sub_type": "flow",
  "index": "0",
  "parameters": [
    {
      "type": "action",
      "action": {
        "flow_token": "customer-123",
        "flow_action_data": {
          "order_id": "ORD-12345"
        }
      }
    }
  ]
}
```

## Complete Example

```json
{
  "to": "6590073708",
  "type": "template",
  "template": {
    "name": "delivery_update",
    "language": {
      "code": "en_US"
    },
    "components": [
      {
        "type": "header",
        "parameters": [
          {
            "type": "image",
            "image": {
              "link": "https://example-bucket.oss-ap-southeast-1.aliyuncs.com/delivery.jpg"
            }
          }
        ]
      },
      {
        "type": "body",
        "parameters": [
          { "type": "text", "text": "Jane" },
          { "type": "text", "text": "ORD-12345" },
          {
            "type": "date_time",
            "date_time": {
              "fallback_value": "9 September 2026, 10:30 AM"
            }
          }
        ]
      },
      {
        "type": "button",
        "sub_type": "url",
        "index": "0",
        "parameters": [
          { "type": "text", "text": "ORD-12345" }
        ]
      },
      {
        "type": "button",
        "sub_type": "quick_reply",
        "index": "1",
        "parameters": [
          { "type": "payload", "payload": "TRACK_DELIVERY:ORD-12345" }
        ]
      }
    ]
  }
}
```

## Constraints

- The backend forwards `parameters` unchanged to Meta. Invalid combinations are rejected by Meta, not locally validated.
- Use exactly the number, order, and type of parameters configured in the approved template.
- A media object must contain either `id` or `link`; do not send both.
- A URL used as media must be accessible to Meta over HTTPS. Use a signed OSS URL only when its expiry is long enough for Meta to fetch it.
- Omit a component that has no runtime parameters.
- Do not send a top-level `FLOW` or `QUICK_REPLY` component type. The valid Meta pattern is `"type": "button"` with a matching `sub_type`.
- `index` values are strings, not integers.
- `flow_action_data` is optional and only used when the Flow requires custom payload fields.
