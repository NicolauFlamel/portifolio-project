# Gestão Orçamentária Pública com Blockchain
Transparência • Auditoria • Rastreabilidade • Imutabilidade

Este repositório contém o sistema desenvolvido como projeto acadêmico para garantir transparência, integridade e auditoria nos dados públicos entre União, Estados e Regiões (agrupamentos de municípios), utilizando Hyperledger Fabric e Go (Gin).

## 1. Contexto

A gestão e a fiscalização das despesas públicas brasileiras ainda enfrentam limitações estruturais relacionadas à centralização, falta de garantias criptográficas e problemas de atualização dos dados. Embora o Portal da Transparência consolide informações federais em um sistema centralizado, o próprio governo reconhece que os dados não são atualizados em tempo real, podendo haver defasagem temporal entre a ocorrência dos gastos e sua disponibilização pública. Além disso, o portal não oferece garantias de imutabilidade, já que não utiliza tecnologias como blockchain; os dados podem ser alterados nos sistemas de origem sem que haja um mecanismo público de prova criptográfica que assegure sua integridade histórica
(Fonte: Portal da Transparência do Governo Federal — https://portaldatransparencia.gov.br/despesas/lista-consultas).

O Tribunal de Contas da União reforça esses desafios ao registrar que órgãos públicos frequentemente removem ou deixam de atualizar informações essenciais, prejudicando diretamente a transparência ativa e dificultando o controle social. Segundo o TCU, “órgãos públicos e servidores indevidamente removem ou não mantêm atualizadas informações essenciais para a transparência ativa”, o que compromete a confiabilidade e a completude dos dados disponibilizados à sociedade
(Fonte: TCU — https://portal.tcu.gov.br/imprensa/noticias/falta-de-publicidade-pelos-orgaos-publicos-diminui-transparencia-e-dificulta-controle-social).

Esses problemas evidenciam que, apesar dos avanços institucionais, a arquitetura atual da transparência pública carece de mecanismos robustos de integridade, rastreabilidade e sincronização, sobretudo quando envolve a interação entre União, Estados e Municípios — cada qual operando sistemas próprios. Nesse contexto, uma infraestrutura blockchain pode atuar como uma camada adicional de segurança e confiabilidade, garantindo imutabilidade criptográfica, registro distribuído e verificação independente dos repasses e execuções orçamentárias, fortalecendo o controle social e reduzindo riscos de inconsistências ou manipulação de dados.

## 2. Requisitos Funcionais
### RF001 - Gerenciamento de Tipos de Documento

Descrição: O sistema deve permitir o cadastro, consulta e desativação de tipos de documentos que servem como templates para os registros de gastos.
Critérios de Aceitação:

- Deve ser possível criar um tipo de documento com campos obrigatórios e opcionais
- Cada tipo de documento deve ter ID único, nome e descrição
- Deve ser possível listar todos os tipos de documento de um canal
- Deve ser possível consultar um tipo de documento específico por ID
- Deve ser possível desativar (mas não excluir) um tipo de documento
- Tipos desativados não permitem criação de novos documentos

Regras de Negócio:

- RN001: ID do tipo de documento deve ser único no canal
- RN002: Tipos de documento são específicos de cada canal (não compartilhados)
- RN003: Campos obrigatórios devem estar presentes em todos os documentos deste tipo


### RF002 - Criação de Documentos de Gasto
Descrição: O sistema deve permitir a criação de documentos representando gastos governamentais (contratos, equipamentos, transferências, etc.).
Critérios de Aceitação:

- Deve ser possível criar documento informando tipo, título, valor e dados customizados
- Sistema deve gerar ID único automaticamente se não fornecido
- Sistema deve calcular hash criptográfico (SHA-256) do conteúdo
- Documento deve registrar organização criadora, data e usuário
- Documento deve iniciar com status ACTIVE
- Sistema deve validar campos obrigatórios do tipo de documento

Regras de Negócio:

- RN004: Tipo de documento deve existir e estar ativo
- RN005: Campos obrigatórios do tipo devem estar presentes em data
- RN006: Hash do conteúdo (contentHash) é calculado automaticamente
- RN007: Documento criado em um canal só pode ser modificado pela organização criadora


### RF003 - Consulta de Documentos
Descrição: O sistema deve permitir consultas e filtros sobre os documentos de gasto.
Critérios de Aceitação:

- Deve ser possível consultar documento específico por ID
- Deve ser possível listar documentos com filtros:

  - Por tipo de documento
  - Por status (ACTIVE, INVALIDATED)
  - Por intervalo de datas
  - Por intervalo de valores
  - Por presença de documento vinculado
  - Por direção do vínculo (OUTGOING, INCOMING)

- Consultas devem suportar paginação
- Sistema deve retornar bookmark para próxima página

Regras de Negócio:

- RN008: Qualquer organização pode consultar documentos (transparência)
- RN009: Consultas retornam apenas documentos do canal especificado
- RN010: Paginação padrão de 20 documentos por página


### RF004 - Histórico de Documentos
Descrição: O sistema deve manter e permitir consulta do histórico completo de alterações de um documento.
Critérios de Aceitação:

- Deve ser possível consultar histórico completo de um documento
- Histórico deve incluir todas as versões do documento
- Cada entrada deve conter: ID da transação, timestamp, dados do documento
- Flag isDelete deve indicar se foi operação de exclusão (sempre false neste sistema)

Regras de Negócio:

- RN011: Histórico é imutável e completo (blockchain)
- RN012: Não há operações de exclusão, apenas invalidação
- RN013: Histórico preserva todas as versões anteriores


### RF005 - Invalidação de Documentos
Descrição: O sistema deve permitir invalidar documentos para correção de erros, mantendo registro completo.
Critérios de Aceitação:

- Deve ser possível invalidar documento informando motivo
- Deve ser possível vincular documento de correção
- Sistema deve atualizar status para INVALIDATED
- Sistema deve registrar quem invalidou, quando e por quê
- Documento original deve permanecer visível no blockchain

Regras de Negócio:

- RN014: Apenas organização criadora pode invalidar documento
- RN015: Motivo da invalidação é obrigatório
- RN016: Documento invalidado permanece no blockchain (imutabilidade)
- RN017: Documento de correção pode ser referenciado opcionalmente
- RN018: Consultas padrão podem filtrar documentos invalidados


### RF006 - Transferências Entre Canais (Cross-Channel)
Descrição: O sistema deve permitir registrar transferências de recursos entre níveis governamentais (Federal → Estadual → Municipal).
Critérios de Aceitação:

- Deve ser possível iniciar transferência especificando canal origem, destino e organização destino
- Sistema deve criar documento no canal origem com hash criptográfico
- Documento deve marcar direção como OUTGOING
- Deve ser possível reconhecer transferência no canal destino
- Sistema deve criar documento no canal destino vinculando ao documento origem
- Documento destino deve copiar hash do documento origem (âncora)
- Documento destino deve marcar direção como INCOMING

Regras de Negócio:

- RN019: Transferência cria dois documentos: um no canal origem, outro no destino
- RN020: Hash do documento origem é copiado para linkedDocHash do documento destino
- RN021: Organização de origem aprova sua saída (OUTGOING)
- RN022: Organização de destino aprova sua entrada (INCOMING)
- RN023: Valores devem ser iguais para verificação ser válida
- RN024: Cada organização só pode criar documentos em seu próprio canal


### RF007 - Verificação de Âncoras (Cross-Channel)
Descrição: O sistema deve permitir verificação criptográfica de vínculos entre documentos em canais diferentes.
Critérios de Aceitação:

- Deve ser possível verificar vínculo entre dois documentos
- Sistema deve comparar hashes dos documentos
- Sistema deve validar referências cruzadas (IDs e canais)
- Sistema deve validar valores e moedas
- Sistema deve retornar resultado detalhado da verificação

Regras de Negócio:

- RN025: Verificação compara contentHash do origem com linkedDocHash do destino
- RN026: Verificação valida: hashes, IDs, canais e valores
- RN027: Status VERIFIED se todos os critérios atenderem
- RN028: Status MISMATCH se qualquer critério falhar
- RN029: Motivos de falha devem ser listados claramente


### RF008 - Consulta de Documentos Vinculados
Descrição: O sistema deve permitir consultar um documento junto com seu documento vinculado em outro canal.
Critérios de Aceitação:

- Deve ser possível consultar documento com seu vínculo cross-channel
- Sistema deve retornar documento principal
- Sistema deve retornar documento vinculado (se existir)
- Sistema deve indicar se vínculo está verificado

Regras de Negócio:

- RN030: Se documento não possui vínculo, linkedDocument retorna null
- RN031: Sistema valida vínculo automaticamente ao retornar


### RF009 - Health Check e Monitoramento
Descrição: O sistema deve fornecer endpoint para verificação de saúde.
Critérios de Aceitação:

- Endpoint /health deve retornar status do serviço
- Resposta deve indicar se API está operacional

Regras de Negócio:

- RN032: Endpoint público, sem autenticação
- RN033: Retorna 200 OK se sistema está saudável


### RF010 - Rastreamento de Requisições
Descrição: O sistema deve gerar ID único para cada requisição HTTP para rastreamento.
Critérios de Aceitação:

- Cada requisição deve receber ID único (request_id)
- Request ID deve ser retornado nas respostas de erro
- Request ID deve aparecer nos logs estruturados
- Request ID deve permitir correlação de logs

Regras de Negócio:

- RN034: Request ID gerado no formato timestamp-uuid
- RN035: Todas as respostas de erro incluem request_id
- RN036: Logs estruturados incluem request_id para rastreamento

## 3. Casos de Uso/User Stories

### US001 - Cadastrar Tipo de Documento de Pagamento a Fornecedores

Como administrador do sistema federal

Quero cadastrar um tipo de documento "Pagamento a Fornecedores"

Para que possa registrar contratos com fornecedores seguindo template padrão

Cenário Principal:

- Administrador acessa API do backend federal
- Administrador envia requisição POST /api/union/document-types
- Sistema valida campos obrigatórios
- Sistema cria tipo de documento no canal union-channel
- Sistema retorna ID do tipo criado
- Tipo de documento fica disponível para criação de documentos

Critérios de Aceitação:

- Tipo criado com ID "contractor-payment"
- Campos obrigatórios: vendor, contractNumber
- Campos opcionais: invoiceNumber, taxId
- Tipo ativo para uso imediato

Regra de Negócio Aplicada: RN001, RN002

### US002 - Registrar Pagamento a Fornecedor
Como operador financeiro federal

Quero registrar pagamento de R$ 250.000 ao fornecedor Tech Solutions

Para que a transação fique registrada de forma imutável no blockchain

Cenário Principal:

- Operador acessa API do backend federal
- Operador envia requisição POST /api/union/documents com dados do pagamento
- Sistema valida tipo de documento existe e está ativo
- Sistema valida campos obrigatórios (vendor, contractNumber)
- Sistema gera hash SHA-256 do conteúdo
- Sistema cria documento no union-channel
- Sistema retorna ID e hash do documento
- Documento fica disponível para consulta

Critérios de Aceitação:

- Documento criado com status ACTIVE
- ContentHash gerado automaticamente
- Campos vendor e contractNumber presentes
- Registro inclui createdBy (Admin@union.gov.br)
- Documento não pode ser excluído

Regra de Negócio Aplicada: RN004, RN005, RN006, RN007

### US003 - Consultar Contratos de Alto Valor
Como auditor

Quero consultar todos os contratos acima de R$ 200.000

Para que possa analisar gastos de alto valor

Cenário Principal:

- Auditor acessa API pública
- Auditor envia requisição GET /api/union/documents?minAmount=200000
- Sistema busca documentos no blockchain que atendem filtro
- Sistema retorna lista de documentos ordenados
- Sistema inclui bookmark para paginação

Critérios de Aceitação:

- Apenas documentos com amount >= 200000 retornados
- Documentos incluem todos os dados (tipo, valor, vendor, etc)
- Paginação com 20 documentos por padrão
- Bookmark permite buscar próxima página

Regra de Negócio Aplicada: RN008, RN009, RN010

### US004 - Corrigir Erro em Documento
Como administrador federal

Quero corrigir documento com valor errado (R$ 500K em vez de R$ 550K)

Para que o valor correto fique registrado mantendo histórico da correção

Cenário Principal:

- Administrador cria novo documento com valor correto (R$ 550K)
- Sistema retorna ID do documento de correção
- Administrador envia requisição POST /api/union/documents/{id-errado}/invalidate
- Administrador informa motivo e ID do documento de correção
- Sistema valida que apenas organização criadora pode invalidar
- Sistema atualiza documento original para INVALIDATED
- Sistema registra motivo, data, usuário e link para correção
- Documento original permanece visível com status INVALIDATED

Critérios de Aceitação:

- Documento original permanece no blockchain
- Status alterado para INVALIDATED
- Motivo da invalidação registrado
- Link para documento de correção presente
- Timestamp e usuário que invalidou registrados
- Histórico completo preservado

Regra de Negócio Aplicada: RN014, RN015, RN016, RN017

### US005 - Transferir Verba Federal para Estado
Como administrador federal

Quero transferir R$ 10 milhões para o estado de São Paulo (educação)

Para que o repasse fique registrado em ambos os canais de forma verificável

Cenário Principal - Parte 1: Federal Inicia:

- Administrador federal acessa API federal
- Administrador envia POST /api/transfers/initiate
- Sistema valida campos obrigatórios
- Sistema cria documento no union-channel
- Sistema calcula contentHash do documento
- Sistema marca linkedDirection como OUTGOING
- Sistema retorna ID e hash do documento criado

Cenário Principal - Parte 2: Estado Reconhece:

- Administrador estadual consulta documento federal via API
- Administrador estadual copia ID e hash do documento federal
- Administrador envia POST /api/state/transfers/acknowledge
- Sistema cria documento no state-channel
- Sistema preenche linkedDocId com ID federal
- Sistema preenche linkedDocHash com hash federal (ÂNCORA)
- Sistema marca linkedDirection como INCOMING
- Sistema retorna ID do documento estadual

Cenário Principal - Parte 3: Verificação:

- Qualquer interessado envia POST /api/anchors/verify
- Sistema busca documento federal no union-channel
- Sistema busca documento estadual no state-channel
- Sistema compara contentHash federal com linkedDocHash estadual
- Sistema compara valores, IDs e canais
- Sistema retorna resultado VERIFIED

Critérios de Aceitação:

- Dois documentos criados (um em cada canal)
- Hash do federal copiado para campo linkedDocHash do estadual
- Valores iguais em ambos os documentos
- Verificação retorna status VERIFIED
- Qualquer pessoa pode verificar a âncora

Regra de Negócio Aplicada: RN019, RN020, RN021, RN022, RN023, RN024, RN025

### US006 - Auditor Verifica Inconsistência em Transferência
Como auditor independente

Quero verificar se transferência federal foi corretamente reconhecida pelo estado

Para que possa identificar discrepâncias ou fraudes

Cenário Principal - Transferência Correta:

- Auditor identifica documento de transferência federal
- Auditor identifica documento de reconhecimento estadual
- Auditor envia POST /api/anchors/verify com ambos os IDs
- Sistema verifica hashes são idênticos
- Sistema verifica valores são iguais
- Sistema retorna status VERIFIED

Cenário Alternativo - Valores Divergentes:

- Estado registra valor diferente do federal
- Auditor envia verificação
- Sistema detecta valores diferentes (10M vs 15M)
- Sistema retorna status MISMATCH
- Sistema lista motivo: "Amount mismatch"
- Discrepância fica exposta publicamente

Critérios de Aceitação:

- Verificação identifica quando hashes não coincidem
- Verificação identifica quando valores diferem
- Motivos de falha são listados claramente
- Resultado é público e verificável por qualquer um

Regra de Negócio Aplicada: RN025, RN026, RN027, RN028, RN029

### US007 - Consultar Histórico Completo de Documento
Como investigador

Quero ver todo o histórico de alterações de um documento

Para que possa rastrear correções e mudanças de status

Cenário Principal:

- Investigador identifica ID do documento
- Investigador envia GET /api/union/documents/{id}/history
- Sistema busca histórico completo no blockchain
- Sistema retorna todas as versões do documento
- Cada versão inclui: timestamp, ID da transação, dados do documento

Critérios de Aceitação:

- Histórico inclui criação original
- Histórico inclui invalidação (se houver)
- Histórico em ordem cronológica
- Cada entrada tem timestamp e ID de transação únicos
- Flag isDelete sempre false (não há exclusões)

Regra de Negócio Aplicada: RN011, RN012, RN013

### US008 - Estado Consulta Documento Federal para Reconhecimento
Como administrador estadual

Quero consultar documento de transferência federal

Para que possa copiar seu hash e criar reconhecimento

Cenário Principal:

Administrador estadual recebe notificação de transferência federal
Administrador envia GET http://federal-api:3000/api/union/documents/{transfer-id}
API federal retorna documento completo
Documento inclui contentHash: "9f86d081884c..."
Administrador usa esse hash em seu reconhecimento

Critérios de Aceitação:

- Estado pode ler documentos do canal federal
- Documento retorna contentHash calculado
- Hash pode ser copiado para linkedDocHash
- Transparência entre níveis governamentais

Regra de Negócio Aplicada: RN008, RN023
