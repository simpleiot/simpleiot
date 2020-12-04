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
    , node : Node
    , onApiDelete : String -> msg
    , onEditNodePoint : String -> Point -> msg
    , onDiscardEdits : msg
    , onApiPostPoints : String -> msg
    }
    -> Element msg
view o =
    let
        textInput =
            Form.nodeTextInput { onEditNodePoint = o.onEditNodePoint, node = o.node, now = o.now }

        numberInput =
            Form.nodeNumberInput { onEditNodePoint = o.onEditNodePoint, node = o.node, now = o.now }

        optionInput =
            Form.nodeOptionInput { onEditNodePoint = o.onEditNodePoint, node = o.node, now = o.now }

        clientServer =
            Point.getPointText o.node.points Point.typeClientServer
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
                Point.getPointText o.node.points Point.typeDescription
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
                    , numberInput Point.typeDebug "Debug level (0-9)"
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
