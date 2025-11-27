# Gestão Orçamentária Pública com Blockchain
Transparência • Auditoria • Rastreabilidade • Imutabilidade

Este repositório contém o sistema desenvolvido como projeto acadêmico para garantir transparência, integridade e auditoria nos dados públicos entre União, Estados e Regiões (agrupamentos de municípios), utilizando Hyperledger Fabric e Go (Gin).

## Contexto

A gestão e a fiscalização das despesas públicas brasileiras ainda enfrentam limitações estruturais relacionadas à centralização, falta de garantias criptográficas e problemas de atualização dos dados. Embora o Portal da Transparência consolide informações federais em um sistema centralizado, o próprio governo reconhece que os dados não são atualizados em tempo real, podendo haver defasagem temporal entre a ocorrência dos gastos e sua disponibilização pública. Além disso, o portal não oferece garantias de imutabilidade, já que não utiliza tecnologias como blockchain; os dados podem ser alterados nos sistemas de origem sem que haja um mecanismo público de prova criptográfica que assegure sua integridade histórica
(Fonte: Portal da Transparência do Governo Federal — https://portaldatransparencia.gov.br/despesas/lista-consultas).

O Tribunal de Contas da União reforça esses desafios ao registrar que órgãos públicos frequentemente removem ou deixam de atualizar informações essenciais, prejudicando diretamente a transparência ativa e dificultando o controle social. Segundo o TCU, “órgãos públicos e servidores indevidamente removem ou não mantêm atualizadas informações essenciais para a transparência ativa”, o que compromete a confiabilidade e a completude dos dados disponibilizados à sociedade
(Fonte: TCU — https://portal.tcu.gov.br/imprensa/noticias/falta-de-publicidade-pelos-orgaos-publicos-diminui-transparencia-e-dificulta-controle-social).

Esses problemas evidenciam que, apesar dos avanços institucionais, a arquitetura atual da transparência pública carece de mecanismos robustos de integridade, rastreabilidade e sincronização, sobretudo quando envolve a interação entre União, Estados e Municípios — cada qual operando sistemas próprios. Nesse contexto, uma infraestrutura blockchain pode atuar como uma camada adicional de segurança e confiabilidade, garantindo imutabilidade criptográfica, registro distribuído e verificação independente dos repasses e execuções orçamentárias, fortalecendo o controle social e reduzindo riscos de inconsistências ou manipulação de dados.

## Requisitos Funcionais
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
