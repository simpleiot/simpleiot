module Components.NodeModbus exposing (view)

import Api.Node exposing (Node)
import Api.Point as Point exposing (Point)
import Element exposing (..)
import Element.Border as Border
import Time
import UI.Form as Form
import UI.Icon as Icon
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view :
    { isRoot : Bool
    , now : Time.Posix
    , zone : Time.Zone
    , modified : Bool
    , expDetail : Bool
    , parent : Maybe Node
    , node : Node
    , onApiDelete : String -> msg
    , onEditNodePoint : String -> Point -> msg
    , onDiscardEdits : msg
    , onApiPostPoints : String -> msg
    }
    -> Element msg
view o =
    let
        labelWidth =
            180

        textInput =
            Form.nodeTextInput
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , labelWidth = labelWidth
                }

        numberInput =
            Form.nodeNumberInput
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , labelWidth = labelWidth
                }

        counterWithReset =
            Form.nodeCounterWithReset
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , labelWidth = labelWidth
                }

        optionInput =
            Form.nodeOptionInput
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , labelWidth = labelWidth
                }

        clientServer =
            Point.getText o.node.points Point.typeClientServer
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.bus
            , text <|
                Point.getText o.node.points Point.typeDescription
            ]
            :: (if o.expDetail then
                    [ textInput Point.typeDescription "Description"
                    , optionInput Point.typeClientServer
                        "Client/Server"
                        [ ( Point.valueClient, "client" )
                        , ( Point.valueServer, "server" )
                        ]
                    , textInput Point.typePort "Port"
                    , textInput Point.typeBaud "Baud"
                    , viewIf (clientServer == Point.valueServer) <|
                        numberInput Point.typeID "Device ID"
                    , numberInput Point.typePollPeriod "Poll period (ms)"
                    , numberInput Point.typeDebug "Debug level (0-9)"
                    , counterWithReset Point.typeErrorCount Point.typeErrorCountReset "Error Count"
                    , counterWithReset Point.typeErrorCountEOF Point.typeErrorCountEOFReset "EOF Error Count"
                    , counterWithReset Point.typeErrorCountCRC Point.typeErrorCountCRCReset "CRC Error Count"
                    , viewIf o.modified <|
                        Form.buttonRow
                            [ Form.button
                                { label = "save"
                                , color = colors.blue
                                , onPress = o.onApiPostPoints o.node.id
                                }
                            , Form.button
                                { label = "discard"
                                , color = colors.gray
                                , onPress = o.onDiscardEdits
                                }
                            ]
                    ]

                else
                    []
               )
