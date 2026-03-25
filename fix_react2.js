const fs = require('fs');
const content = fs.readFileSync('web/frontend/src/components/models/edit-model-sheet.tsx', 'utf8');
const fixed = content.replace(/placeholder='\{"X-My-Header": "value"\}'/, `placeholder='{"X-My-Header": "value"}'
                />
              </Field>
              <Field`);
fs.writeFileSync('web/frontend/src/components/models/edit-model-sheet.tsx', fixed);
